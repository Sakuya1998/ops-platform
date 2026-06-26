package config

import "github.com/ops-platform/pkg/config"

type Config struct {
	Server   config.ServerConfig   `mapstructure:"server"`
	Database config.DatabaseConfig `mapstructure:"database"`
	Redis    config.RedisConfig    `mapstructure:"redis"`
	Kafka    config.KafkaConfig    `mapstructure:"kafka"`
	Consul   config.ConsulConfig   `mapstructure:"consul"`
	JWT      config.JWTConfig      `mapstructure:"jwt"`
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	_ = config.LoadConfig("config.yaml", cfg)
	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	config.ApplyServerEnv(&cfg.Server)
	config.ApplyDatabaseEnv(&cfg.Database)
	config.ApplyRedisEnv(&cfg.Redis)
	config.ApplyKafkaEnv(&cfg.Kafka)
	config.ApplyConsulEnv(&cfg.Consul)
	config.ApplyJWTEnv(&cfg.JWT)
}

func DefaultConfig() *Config {
	return &Config{
		Server: config.ServerConfig{Name: "iam-svc", HTTPPort: 8081, GRPCPort: 9081},
		Database: config.DatabaseConfig{Host: "localhost", Port: 5432, User: "opsadmin",
			Password: "ops@2026", DBName: "iam_svc", SSLMode: "disable"},
		Redis:  config.RedisConfig{Host: "localhost", Port: 6379, Password: "", DB: 0},
		Kafka:  config.KafkaConfig{Brokers: []string{"localhost:9092"}, Topic: "ops-platform-events"},
		Consul: config.ConsulConfig{Address: "localhost:8500"},
		JWT:    config.JWTConfig{Secret: "ops-platform-jwt-secret-2026", ExpireHour: 2, Issuer: "ops-platform"},
	}
}
