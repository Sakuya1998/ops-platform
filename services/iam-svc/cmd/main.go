package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/config"
	iamgrpc "github.com/Sakuya1998/ops-platform/services/iam-svc/internal/grpc"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/handler"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/router"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/service"
	"github.com/Sakuya1998/ops-platform/pkg/cache"
	"github.com/Sakuya1998/ops-platform/pkg/consul"
	"github.com/Sakuya1998/ops-platform/pkg/database"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
	"github.com/Sakuya1998/ops-platform/pkg/logger"
	iamv1 "github.com/Sakuya1998/ops-platform/pkg/proto/iam/v1"
	"github.com/Sakuya1998/ops-platform/pkg/trace"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	logger.Init("iam-svc")
	trace.Init("iam-svc")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir, err := database.ResolveMigrationsDir("services/iam-svc/migrations", "migrations")
	if err != nil {
		log.Fatalf("Failed to resolve migrations dir: %v", err)
	}
	migrationResult, err := database.RunMigrations(context.Background(), db, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	database.LogMigrationResult("iam-svc", migrationsDir, migrationResult)

	consulClient, err := consul.NewClient(cfg.Consul.Address)
	if err != nil {
		log.Printf("[WARN] Failed to create Consul client: %v", err)
	}

	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer kafkaProducer.Close()

	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	apiPermRepo := repository.NewAPIPermissionRepository(db)

	permissionCache := newPermissionCache(cfg)
	iamSvc := service.NewIAMService(roleRepo, permRepo, kafkaProducer, cfg.JWT).
		WithAPIPermissionRepository(apiPermRepo).
		WithPermissionCache(permissionCache)
	iamHandler := handler.NewIAMHandler(iamSvc)

	consumer := service.NewIAMConsumer(roleRepo).WithIAMService(iamSvc)
	go consumer.Start(context.Background(), cfg.Kafka.Brokers, cfg.Kafka.Topic, "iam-svc-group")

	go startGRPCServer(cfg, iamSvc)
	go startHTTPServer(cfg, iamHandler, consulClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Get().Info("Shutting down iam-svc...")
}

func newPermissionCache(cfg *config.Config) *cache.Cache {
	redisBackend := cache.NewRedisBackendFromOptions(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	return cache.New(cache.Options{
		DefaultTTL:     5 * time.Minute,
		MaxEntries:     10000,
		L2:             redisBackend,
		IgnoreL2Errors: true,
	})
}

func startGRPCServer(cfg *config.Config, iamSvc *service.IAMService) {
	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr())
	if err != nil {
		log.Fatalf("IAM gRPC listen error: %v", err)
	}
	server := grpc.NewServer()
	iamv1.RegisterIAMServiceServer(server, iamgrpc.NewServer(iamSvc))
	logger.Get().Info(fmt.Sprintf("IAM service gRPC server listening on %s", cfg.Server.GRPCAddr()))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("IAM gRPC server error: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, h *handler.IAMHandler, cc *consul.Client) {
	r := router.New(h)
	if cc != nil {
		advertiseAddress := cfg.Server.AdvertiseAddress()
		reg := consul.Registration{
			ID: "iam-svc-" + advertiseAddress, Name: "iam-svc",
			Address: advertiseAddress, Port: cfg.Server.HTTPPort, Tags: []string{"iam", "v1"},
		}
		if err := cc.Register(reg); err != nil {
			logger.Get().Warn(fmt.Sprintf("Consul registration failed: %v", err))
		}
	}

	addr := cfg.Server.HTTPAddr()
	logger.Get().Info(fmt.Sprintf("IAM service HTTP server listening on %s", addr))
	if err := r.Run(addr); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
