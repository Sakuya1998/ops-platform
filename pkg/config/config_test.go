package config

import "testing"

func TestApplyServerEnvUsesServiceAddress(t *testing.T) {
	t.Setenv("OPS_SERVICE_ADDRESS", "bff-api")

	cfg := ServerConfig{}
	ApplyServerEnv(&cfg)

	if cfg.Address != "bff-api" {
		t.Fatalf("expected OPS_SERVICE_ADDRESS to set server address, got %q", cfg.Address)
	}
}

func TestApplyServerEnvLetsServerAddressOverrideServiceAddress(t *testing.T) {
	t.Setenv("OPS_SERVICE_ADDRESS", "bff-api")
	t.Setenv("OPS_SERVER_ADDRESS", "custom-bff")

	cfg := ServerConfig{}
	ApplyServerEnv(&cfg)

	if cfg.Address != "custom-bff" {
		t.Fatalf("expected OPS_SERVER_ADDRESS to override OPS_SERVICE_ADDRESS, got %q", cfg.Address)
	}
}

func TestAdvertiseAddressUsesConfiguredAddress(t *testing.T) {
	cfg := ServerConfig{Address: "auth-svc"}

	if got := cfg.AdvertiseAddress(); got != "auth-svc" {
		t.Fatalf("expected configured advertise address, got %q", got)
	}
}
