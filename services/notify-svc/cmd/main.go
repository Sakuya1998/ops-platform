package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/config"
	notifygrpc "github.com/Sakuya1998/ops-platform/services/notify-svc/internal/grpc"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/handler"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/router"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/service"
	"github.com/Sakuya1998/ops-platform/pkg/consul"
	"github.com/Sakuya1998/ops-platform/pkg/database"
	"github.com/Sakuya1998/ops-platform/pkg/logger"
	notifyv1 "github.com/Sakuya1998/ops-platform/pkg/proto/notify/v1"
	"github.com/Sakuya1998/ops-platform/pkg/trace"
	"google.golang.org/grpc"
)

func main() {
	logger.Init("notify-svc")
	trace.Init("notify-svc")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir, err := database.ResolveMigrationsDir("services/notify-svc/migrations", "migrations")
	if err != nil {
		log.Fatalf("Failed to resolve migrations dir: %v", err)
	}
	migrationResult, err := database.RunMigrations(context.Background(), db, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	database.LogMigrationResult("notify-svc", migrationsDir, migrationResult)

	consulClient, err := consul.NewClient(cfg.Consul.Address)
	if err != nil {
		log.Printf("[WARN] Failed to create Consul client: %v", err)
	}

	channelRepo := repository.NewChannelRepository(db)
	tmplRepo := repository.NewTemplateRepository(db)
	logRepo := repository.NewLogRepository(db)

	notifySvc := service.NewNotifyService(channelRepo, tmplRepo, logRepo)
	notifyHandler := handler.NewNotifyHandler(channelRepo, tmplRepo, logRepo)

	notifyConsumer := service.NewNotifyConsumer(notifySvc)
	go notifyConsumer.Start(context.Background(), cfg.Kafka.Brokers, cfg.Kafka.Topic, "notify-svc-group")

	go startGRPCServer(cfg, channelRepo, tmplRepo, logRepo, notifySvc)
	go startHTTPServer(cfg, notifyHandler, consulClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Get().Info("Shutting down notify-svc...")
}

func startGRPCServer(
	cfg *config.Config,
	channelRepo *repository.ChannelRepository,
	tmplRepo *repository.TemplateRepository,
	logRepo *repository.LogRepository,
	notifySvc *service.NotifyService,
) {
	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr())
	if err != nil {
		log.Fatalf("Notify gRPC listen error: %v", err)
	}
	server := grpc.NewServer()
	notifyv1.RegisterNotifyServiceServer(server, notifygrpc.NewServer(channelRepo, tmplRepo, logRepo, notifySvc))
	logger.Get().Info(fmt.Sprintf("Notify service gRPC server listening on %s", cfg.Server.GRPCAddr()))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Notify gRPC server error: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, h *handler.NotifyHandler, cc *consul.Client) {
	r := router.New(h)

	if cc != nil {
		advertiseAddress := cfg.Server.AdvertiseAddress()
		reg := consul.Registration{
			ID: "notify-svc-" + advertiseAddress, Name: "notify-svc",
			Address: advertiseAddress, Port: cfg.Server.HTTPPort, Tags: []string{"notify", "v1"},
		}
		if err := cc.Register(reg); err != nil {
			logger.Get().Warn(fmt.Sprintf("Consul registration failed: %v", err))
		}
	}
	logger.Get().Info(fmt.Sprintf("Notify service HTTP server on %s", cfg.Server.HTTPAddr()))
	if err := r.Run(cfg.Server.HTTPAddr()); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
