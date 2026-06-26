package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig(path string, cfg interface{}) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvPrefix("OPS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if _, statErr := os.Stat(path); statErr == nil {
				return fmt.Errorf("read config: %w", err)
			}
		}
	}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
}

type ConsulConfig struct {
	Address string `mapstructure:"address"`
}

type ServerConfig struct {
	Name     string `mapstructure:"name"`
	Address  string `mapstructure:"address"`
	HTTPPort int    `mapstructure:"http_port"`
	GRPCPort int    `mapstructure:"grpc_port"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireHour int    `mapstructure:"expire_hour"`
	Issuer     string `mapstructure:"issuer"`
}

func ApplyDatabaseEnv(cfg *DatabaseConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_DB_HOST"); value != "" {
		cfg.Host = value
	}
	if value := os.Getenv("OPS_DB_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Port = port
		}
	}
	if value := os.Getenv("OPS_DB_USER"); value != "" {
		cfg.User = value
	}
	if value := os.Getenv("OPS_DB_PASSWORD"); value != "" {
		cfg.Password = value
	}
	if value := os.Getenv("OPS_DB_NAME"); value != "" {
		cfg.DBName = value
	}
	if value := os.Getenv("OPS_DB_SSLMODE"); value != "" {
		cfg.SSLMode = value
	}
}

func ApplyRedisEnv(cfg *RedisConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_REDIS_HOST"); value != "" {
		cfg.Host = value
	}
	if value := os.Getenv("OPS_REDIS_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			cfg.Port = port
		}
	}
	if value := os.Getenv("OPS_REDIS_PASSWORD"); value != "" {
		cfg.Password = value
	}
	if value := os.Getenv("OPS_REDIS_DB"); value != "" {
		if db, err := strconv.Atoi(value); err == nil {
			cfg.DB = db
		}
	}
	if value := os.Getenv("OPS_REDIS_ADDR"); value != "" {
		parts := strings.Split(value, ":")
		if len(parts) > 0 && parts[0] != "" {
			cfg.Host = parts[0]
		}
		if len(parts) > 1 {
			if port, err := strconv.Atoi(parts[1]); err == nil {
				cfg.Port = port
			}
		}
	}
}

func ApplyKafkaEnv(cfg *KafkaConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_KAFKA_BROKERS"); value != "" {
		cfg.Brokers = strings.Split(value, ",")
	}
	if value := os.Getenv("OPS_KAFKA_TOPIC"); value != "" {
		cfg.Topic = value
	}
}

func ApplyConsulEnv(cfg *ConsulConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_CONSUL_ADDR"); value != "" {
		cfg.Address = value
	}
}

func ApplyServerEnv(cfg *ServerConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_SERVICE_ADDRESS"); value != "" {
		cfg.Address = value
	}
	if value := os.Getenv("OPS_SERVER_ADDRESS"); value != "" {
		cfg.Address = value
	}
}

func ApplyJWTEnv(cfg *JWTConfig) {
	if cfg == nil {
		return
	}
	if value := os.Getenv("OPS_JWT_SECRET"); value != "" {
		cfg.Secret = value
	}
	if value := os.Getenv("OPS_JWT_EXPIRE_HOUR"); value != "" {
		if hours, err := strconv.Atoi(value); err == nil {
			cfg.ExpireHour = hours
		}
	}
	if value := os.Getenv("OPS_JWT_ISSUER"); value != "" {
		cfg.Issuer = value
	}
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func (s *ServerConfig) GRPCAddr() string {
	return fmt.Sprintf(":%d", s.GRPCPort)
}

func (s *ServerConfig) HTTPAddr() string {
	return fmt.Sprintf(":%d", s.HTTPPort)
}

func (s *ServerConfig) AdvertiseAddress() string {
	if s.Address != "" {
		return s.Address
	}
	hostname, _ := os.Hostname()
	return hostname
}
