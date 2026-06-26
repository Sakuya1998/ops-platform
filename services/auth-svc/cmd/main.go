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

	"github.com/google/uuid"
	"github.com/ops-platform/auth-svc/internal/config"
	authgrpc "github.com/ops-platform/auth-svc/internal/grpc"
	"github.com/ops-platform/auth-svc/internal/handler"
	"github.com/ops-platform/auth-svc/internal/model"
	"github.com/ops-platform/auth-svc/internal/repository"
	"github.com/ops-platform/auth-svc/internal/router"
	"github.com/ops-platform/auth-svc/internal/service"
	"github.com/ops-platform/pkg/cache"
	"github.com/ops-platform/pkg/consul"
	secretcrypto "github.com/ops-platform/pkg/crypto"
	"github.com/ops-platform/pkg/database"
	"github.com/ops-platform/pkg/kafka"
	"github.com/ops-platform/pkg/logger"
	authv1 "github.com/ops-platform/pkg/proto/auth/v1"
	"github.com/ops-platform/pkg/trace"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	logger.Init("auth-svc")
	trace.Init("auth-svc")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	migrationsDir, err := database.ResolveMigrationsDir("services/auth-svc/migrations", "migrations")
	if err != nil {
		log.Fatalf("Failed to resolve migrations dir: %v", err)
	}
	migrationResult, err := database.RunMigrations(context.Background(), db, migrationsDir)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	database.LogMigrationResult("auth-svc", migrationsDir, migrationResult)

	orgRepo := repository.NewOrganizationRepository(db)
	ensureDefaultOrganization(orgRepo)

	consulClient, err := consul.NewClient(cfg.Consul.Address)
	if err != nil {
		log.Printf("[WARN] Failed to create Consul client: %v", err)
	}

	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic)
	defer kafkaProducer.Close()

	userRepo := repository.NewUserRepository(db)
	providerRepo := repository.NewProviderRepository(db)

	iamClient := service.NewIAMClient(cfg.IAM.BaseURL)
	ldapSvc := service.NewLdapService(&cfg.LDAP, userRepo)
	secretBox := secretcrypto.NewSecretBox(cfg.Encryption.Secret)
	authSvc := service.NewAuthService(userRepo, providerRepo, orgRepo, iamClient, cfg.JWT, kafkaProducer, ldapSvc, secretBox)
	redisBackend := newRedisBackend(cfg)
	authSvc.WithLoginLimiter(service.NewLoginLimiter(redisBackend, service.LoginLimiterOptions{
		MaxAttempts: cfg.Security.LoginLimitMaxAttempts,
		Window:      cfg.Security.LoginLimitWindow,
	}))
	stateCache := newStateCache(redisBackend)
	oidcSvc := service.NewOIDCServiceWithStateCache(&cfg.OIDC, authSvc, userRepo, providerRepo, secretBox, stateCache)
	authHandler := handler.NewAuthHandler(authSvc, oidcSvc)

	go startGRPCServer(cfg, authSvc, oidcSvc)
	go startHTTPServer(cfg, authHandler, consulClient)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Get().Info("Shutting down auth-svc...")
}

func newRedisBackend(cfg *config.Config) *cache.RedisBackend {
	return cache.NewRedisBackendFromOptions(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func newStateCache(redisBackend *cache.RedisBackend) *cache.Cache {
	return cache.New(cache.Options{
		DefaultTTL:     10 * time.Minute,
		MaxEntries:     10000,
		L2:             redisBackend,
		IgnoreL2Errors: true,
	})
}

func startGRPCServer(cfg *config.Config, authSvc *service.AuthService, oidcSvc *service.OIDCService) {
	listener, err := net.Listen("tcp", cfg.Server.GRPCAddr())
	if err != nil {
		log.Fatalf("Auth gRPC listen error: %v", err)
	}
	server := grpc.NewServer()
	authv1.RegisterAuthServiceServer(server, authgrpc.NewServer(authSvc, oidcSvc))
	logger.Get().Info(fmt.Sprintf("Auth service gRPC server listening on %s", cfg.Server.GRPCAddr()))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Auth gRPC server error: %v", err)
	}
}

func startHTTPServer(cfg *config.Config, h *handler.AuthHandler, cc *consul.Client) {
	r := router.New(h)

	if cc != nil {
		advertiseAddress := cfg.Server.AdvertiseAddress()
		reg := consul.Registration{
			ID:      "auth-svc-" + advertiseAddress,
			Name:    "auth-svc",
			Address: advertiseAddress,
			Port:    cfg.Server.HTTPPort,
			Tags:    []string{"auth", "v1"},
		}
		if err := cc.Register(reg); err != nil {
			logger.Get().Warn(fmt.Sprintf("Consul registration failed: %v", err))
		}
	}

	addr := cfg.Server.HTTPAddr()
	logger.Get().Info(fmt.Sprintf("HTTP server listening on %s", addr))
	if err := r.Run(addr); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func ensureDefaultOrganization(repo *repository.OrganizationRepository) {
	if _, err := repo.GetByCode("default"); err == nil {
		return
	}
	defaultOrg := &model.Organization{
		ID:          uuidMustParse("00000000-0000-0000-0000-000000000001"),
		Name:        "Default Organization",
		Code:        "default",
		Description: "Default organization created automatically",
		Status:      "active",
	}
	if err := repo.Create(defaultOrg); err != nil {
		logger.Get().Warn(fmt.Sprintf("Failed to create default org: %v", err))
	}
}

func uuidMustParse(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}
