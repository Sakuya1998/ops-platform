package config

import (
	"testing"
	"time"
)

func TestDefaultConfigIncludesLoginLimiter(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Security.LoginLimitMaxAttempts != 10 {
		t.Fatalf("expected default max attempts 10, got %d", cfg.Security.LoginLimitMaxAttempts)
	}
	if cfg.Security.LoginLimitWindow != 15*time.Minute {
		t.Fatalf("expected default window 15m, got %v", cfg.Security.LoginLimitWindow)
	}
}

func TestApplyEnvOverridesLoginLimiter(t *testing.T) {
	t.Setenv("OPS_AUTH_LOGIN_LIMIT_MAX_ATTEMPTS", "3")
	t.Setenv("OPS_AUTH_LOGIN_LIMIT_WINDOW_SECONDS", "120")

	cfg := DefaultConfig()
	applyEnv(cfg)

	if cfg.Security.LoginLimitMaxAttempts != 3 {
		t.Fatalf("expected max attempts 3, got %d", cfg.Security.LoginLimitMaxAttempts)
	}
	if cfg.Security.LoginLimitWindow != 2*time.Minute {
		t.Fatalf("expected window 2m, got %v", cfg.Security.LoginLimitWindow)
	}
}
