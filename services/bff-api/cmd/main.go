package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/config"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/handler"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/router"
	"github.com/Sakuya1998/ops-platform/pkg/consul"
	"github.com/Sakuya1998/ops-platform/pkg/logger"
	"github.com/Sakuya1998/ops-platform/pkg/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger.Init("bff-api")
	trace.Init("bff-api")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	consulClient, err := consul.NewClient(cfg.Consul.Address)
	if err != nil {
		log.Printf("[WARN] Failed to create Consul client: %v", err)
	}

	bootstrapHandler := handler.NewBootstrapHandler(cfg.Server.Name)
	authHTTPFallback, err := client.NewAuthHTTPClient(cfg.Services.AuthBaseURL, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}
	authConn, err := grpc.Dial(cfg.Services.AuthGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect auth gRPC: %v", err)
	}
	defer authConn.Close()
	authClient := client.NewAuthGRPCClient(authConn, authHTTPFallback)
	iamConn, err := grpc.Dial(cfg.Services.IAMGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect IAM gRPC: %v", err)
	}
	defer iamConn.Close()
	iamClient := client.NewIAMGRPCClient(iamConn)
	auditConn, err := grpc.Dial(cfg.Services.AuditGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect audit gRPC: %v", err)
	}
	defer auditConn.Close()
	auditClient := client.NewAuditGRPCClient(auditConn)
	notifyConn, err := grpc.Dial(cfg.Services.NotifyGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect notify gRPC: %v", err)
	}
	defer notifyConn.Close()
	notifyClient := client.NewNotifyGRPCClient(notifyConn)
	deps := router.Dependencies{
		Bootstrap:     bootstrapHandler,
		Auth:          handler.NewAuthHandler(authClient),
		IAM:           handler.NewIAMHandler(iamClient),
		Audit:         handler.NewAuditHandler(auditClient),
		Notify:        handler.NewNotifyHandler(notifyClient),
		Permission:    iamClient,
		TokenVerifier: authClient,
	}
	go startHTTPServer(cfg, deps, consulClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Get().Info("Shutting down bff-api...")
}

func startHTTPServer(cfg *config.Config, deps router.Dependencies, cc *consul.Client) {
	r := router.New(deps)

	if cc != nil {
		advertiseAddress := cfg.Server.AdvertiseAddress()
		reg := consul.Registration{
			ID:      "bff-api-" + advertiseAddress,
			Name:    "bff-api",
			Address: advertiseAddress,
			Port:    cfg.Server.HTTPPort,
			Tags:    []string{"bff", "api", "v1"},
		}
		if err := cc.Register(reg); err != nil {
			logger.Get().Warn(fmt.Sprintf("Consul registration failed: %v", err))
		}
	}

	addr := cfg.Server.HTTPAddr()
	logger.Get().Info(fmt.Sprintf("BFF API HTTP server listening on %s", addr))
	if err := r.Run(addr); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
