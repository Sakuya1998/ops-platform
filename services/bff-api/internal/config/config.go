package config

import (
	"os"

	sharedconfig "github.com/ops-platform/pkg/config"
)

type Config struct {
	Server   sharedconfig.ServerConfig `mapstructure:"server"`
	Consul   sharedconfig.ConsulConfig `mapstructure:"consul"`
	Services ServicesConfig            `mapstructure:"services"`
}

type ServicesConfig struct {
	AuthBaseURL    string `mapstructure:"auth_base_url"`
	IAMBaseURL     string `mapstructure:"iam_base_url"`
	AuditBaseURL   string `mapstructure:"audit_base_url"`
	NotifyBaseURL  string `mapstructure:"notify_base_url"`
	AuthGRPCAddr   string `mapstructure:"auth_grpc_addr"`
	IAMGRPCAddr    string `mapstructure:"iam_grpc_addr"`
	AuditGRPCAddr  string `mapstructure:"audit_grpc_addr"`
	NotifyGRPCAddr string `mapstructure:"notify_grpc_addr"`
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	_ = sharedconfig.LoadConfig("config.yaml", cfg)
	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	sharedconfig.ApplyServerEnv(&cfg.Server)
	sharedconfig.ApplyConsulEnv(&cfg.Consul)
	if value := os.Getenv("OPS_AUTH_BASE_URL"); value != "" {
		cfg.Services.AuthBaseURL = value
	}
	if value := os.Getenv("OPS_IAM_BASE_URL"); value != "" {
		cfg.Services.IAMBaseURL = value
	}
	if value := os.Getenv("OPS_AUDIT_BASE_URL"); value != "" {
		cfg.Services.AuditBaseURL = value
	}
	if value := os.Getenv("OPS_NOTIFY_BASE_URL"); value != "" {
		cfg.Services.NotifyBaseURL = value
	}
	if value := os.Getenv("OPS_AUTH_GRPC_ADDR"); value != "" {
		cfg.Services.AuthGRPCAddr = value
	}
	if value := os.Getenv("OPS_IAM_GRPC_ADDR"); value != "" {
		cfg.Services.IAMGRPCAddr = value
	}
	if value := os.Getenv("OPS_AUDIT_GRPC_ADDR"); value != "" {
		cfg.Services.AuditGRPCAddr = value
	}
	if value := os.Getenv("OPS_NOTIFY_GRPC_ADDR"); value != "" {
		cfg.Services.NotifyGRPCAddr = value
	}
}

func DefaultConfig() *Config {
	return &Config{
		Server: sharedconfig.ServerConfig{Name: "bff-api", HTTPPort: 8070, GRPCPort: 9070},
		Consul: sharedconfig.ConsulConfig{Address: "localhost:8500"},
		Services: ServicesConfig{
			AuthBaseURL:    "http://localhost:8080",
			IAMBaseURL:     "http://localhost:8081",
			AuditBaseURL:   "http://localhost:8082",
			NotifyBaseURL:  "http://localhost:8083",
			AuthGRPCAddr:   "localhost:9080",
			IAMGRPCAddr:    "localhost:9081",
			AuditGRPCAddr:  "localhost:9082",
			NotifyGRPCAddr: "localhost:9083",
		},
	}
}
