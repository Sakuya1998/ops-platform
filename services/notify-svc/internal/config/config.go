package config

import "github.com/ops-platform/pkg/config"

type Config struct {
	Server   config.ServerConfig   `mapstructure:"server"`
	Database config.DatabaseConfig `mapstructure:"database"`
	Kafka    config.KafkaConfig    `mapstructure:"kafka"`
	Consul   config.ConsulConfig   `mapstructure:"consul"`
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
	config.ApplyKafkaEnv(&cfg.Kafka)
	config.ApplyConsulEnv(&cfg.Consul)
}

func DefaultConfig() *Config {
	return &Config{
		Server: config.ServerConfig{Name: "notify-svc", HTTPPort: 8083, GRPCPort: 9083},
		Database: config.DatabaseConfig{Host: "localhost", Port: 5432, User: "opsadmin",
			Password: "ops@2026", DBName: "notify_svc", SSLMode: "disable"},
		Kafka:  config.KafkaConfig{Brokers: []string{"localhost:9092"}, Topic: "ops-platform-events"},
		Consul: config.ConsulConfig{Address: "localhost:8500"},
	}
}
