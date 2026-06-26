package config

import (
	"os"
	"strconv"
	"time"

	"github.com/Sakuya1998/ops-platform/pkg/config"
)

type Config struct {
	Server     config.ServerConfig   `mapstructure:"server"`
	Database   config.DatabaseConfig `mapstructure:"database"`
	Redis      config.RedisConfig    `mapstructure:"redis"`
	Kafka      config.KafkaConfig    `mapstructure:"kafka"`
	Consul     config.ConsulConfig   `mapstructure:"consul"`
	JWT        config.JWTConfig      `mapstructure:"jwt"`
	OIDC       OIDCConfig            `mapstructure:"oidc"`
	LDAP       LDAPConfig            `mapstructure:"ldap"`
	IAM        IAMConfig             `mapstructure:"iam"`
	Encryption EncryptionConfig      `mapstructure:"encryption"`
	Security   SecurityConfig        `mapstructure:"security"`
}

type IAMConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type EncryptionConfig struct {
	Secret string `mapstructure:"secret"`
}

type SecurityConfig struct {
	LoginLimitMaxAttempts int           `mapstructure:"login_limit_max_attempts"`
	LoginLimitWindow      time.Duration `mapstructure:"login_limit_window"`
}

type LDAPConfig struct {
	Enabled         bool   `mapstructure:"enabled" json:"enabled"`
	Host            string `mapstructure:"host" json:"host"`
	Port            int    `mapstructure:"port" json:"port"`
	Security        string `mapstructure:"security" json:"security"`
	BindDN          string `mapstructure:"bind_dn" json:"bind_dn"`
	BindPassword    string `mapstructure:"bind_password" json:"bind_password"`
	BaseDN          string `mapstructure:"base_dn" json:"base_dn"`
	UserFilter      string `mapstructure:"user_filter" json:"user_filter"`
	UIDAttr         string `mapstructure:"uid_attr" json:"uid_attr"`
	DisplayNameAttr string `mapstructure:"display_name_attr" json:"display_name_attr"`
	EmailAttr       string `mapstructure:"email_attr" json:"email_attr"`
	AutoProvision   bool   `mapstructure:"auto_provision" json:"auto_provision"`
	DefaultOrgCode  string `mapstructure:"default_org_code" json:"default_org_code"`
	SkipVerify      bool   `mapstructure:"skip_verify" json:"skip_verify"`
}

type OIDCConfig struct {
	Enabled        bool     `mapstructure:"enabled" json:"enabled"`
	ProviderName   string   `mapstructure:"provider_name" json:"provider_name"`
	Issuer         string   `mapstructure:"issuer" json:"issuer"`
	ClientID       string   `mapstructure:"client_id" json:"client_id"`
	ClientSecret   string   `mapstructure:"client_secret" json:"client_secret"`
	RedirectURI    string   `mapstructure:"redirect_uri" json:"redirect_uri"`
	Scopes         []string `mapstructure:"scopes" json:"scopes"`
	AutoProvision  bool     `mapstructure:"auto_provision" json:"auto_provision"`
	DefaultOrgCode string   `mapstructure:"default_org_code" json:"default_org_code"`
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
	if value := os.Getenv("OPS_IAM_BASE_URL"); value != "" {
		cfg.IAM.BaseURL = value
	}
	if value := os.Getenv("OPS_AUTH_ENCRYPTION_SECRET"); value != "" {
		cfg.Encryption.Secret = value
	}
	if value := os.Getenv("OPS_AUTH_LOGIN_LIMIT_MAX_ATTEMPTS"); value != "" {
		if attempts, err := strconv.Atoi(value); err == nil {
			cfg.Security.LoginLimitMaxAttempts = attempts
		}
	}
	if value := os.Getenv("OPS_AUTH_LOGIN_LIMIT_WINDOW_SECONDS"); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			cfg.Security.LoginLimitWindow = time.Duration(seconds) * time.Second
		}
	}
	if cfg.Encryption.Secret == "" {
		cfg.Encryption.Secret = cfg.JWT.Secret
	}
}

func DefaultConfig() *Config {
	return &Config{
		Server: config.ServerConfig{
			Name:     "auth-svc",
			HTTPPort: 8080,
			GRPCPort: 9080,
		},
		Database: config.DatabaseConfig{
			Host: "localhost", Port: 5432, User: "opsadmin",
			Password: "ops@2026", DBName: "auth_svc", SSLMode: "disable",
		},
		Redis: config.RedisConfig{
			Host: "localhost", Port: 6379, Password: "", DB: 0,
		},
		Kafka: config.KafkaConfig{
			Brokers: []string{"localhost:9092"}, Topic: "ops-platform-events",
		},
		Consul: config.ConsulConfig{Address: "localhost:8500"},
		JWT: config.JWTConfig{
			Secret: "ops-platform-jwt-secret-2026", ExpireHour: 2, Issuer: "ops-platform",
		},
		IAM: IAMConfig{BaseURL: "http://localhost:8081"},
		Security: SecurityConfig{
			LoginLimitMaxAttempts: 10,
			LoginLimitWindow:      15 * time.Minute,
		},
	}
}
