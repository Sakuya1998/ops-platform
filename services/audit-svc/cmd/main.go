package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ops-platform/audit-svc/internal/config"
	auditgrpc "github.com/ops-platform/audit-svc/internal/grpc"
	"github.com/ops-platform/audit-svc/internal/handler"
	"github.com/ops-platform/audit-svc/internal/repository"
	"github.com/ops-platform/audit-svc/internal/router"
	"github.com/ops-platform/audit-svc/internal/service"
	"github.com/ops-platform/pkg/consul"
	"github.com/ops-platform/pkg/database"
	"github.com/ops-platform/pkg/logger"
	auditv1 "github.com/ops-platform/pkg/proto/audit/v1"
	"github.com/ops-platform/pkg/trace"
	"google.golang.org/grpc"
)

func main() {
	logger.Init("audit-svc")
	trace.Init("audit-svc")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir, err := database.ResolveMigrationsDir("services/audit-svc/migrations", "migrations")
	if err != nil {
		log.Fatalf("Failed to resolve migrations dir: %v", err)
	}
	migrationResult, err := database.RunMigrations(context.Background(), db, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	database.LogMigrationResult("audit-svc", migrationsDir, migrationResult)

	consulClient, err := consul.NewClient(cfg.Consul.Address)
	if err != nil {
		log.Printf("[WARN] Failed to create Consul client: %v", err)
	}

	auditRepo := repository.NewAuditRepository(db)
	auditHandler := handler.NewAuditHandler(auditRepo)

	consumer := service.NewAuditConsumer(auditRepo)
	go consumer.Start(context.Background(), cfg.Kafka.Brokers, cfg.Kafka.Topic, "audit-svc-group")

	go startGRPCServer(cfg, auditRepo)
	go startHTTPServer(cfg, auditHandler, consulClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Get().Info("Shutting down audit-svc...")
}

func startGRPCServer(cfg *config.Config, repo *repository.AuditRepository) {
	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr())
	if err != nil {
		log.Fatalf("Audit gRPC listen error: %v", err)
	}
	server := grpc.NewServer()
	auditv1.RegisterAuditServiceServer(server, auditgrpc.NewServer(repo))
	logger.Get().Info(fmt.Sprintf("Audit service gRPC server listening on %s", cfg.Server.GRPCAddr()))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Audit gRPC server error: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, h *handler.AuditHandler, cc *consul.Client) {
	r := router.New(h)

	if cc != nil {
		advertiseAddress := cfg.Server.AdvertiseAddress()
		reg := consul.Registration{
			ID: "audit-svc-" + advertiseAddress, Name: "audit-svc",
			Address: advertiseAddress, Port: cfg.Server.HTTPPort, Tags: []string{"audit", "v1"},
		}
		if err := cc.Register(reg); err != nil {
			logger.Get().Warn(fmt.Sprintf("Consul registration failed: %v", err))
		}
	}
	logger.Get().Info(fmt.Sprintf("Audit service HTTP server on %s", cfg.Server.HTTPAddr()))
	if err := r.Run(cfg.Server.HTTPAddr()); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
