package config

import "testing"

func TestDefaultConfigIncludesGRPCServiceAddresses(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Services.AuthGRPCAddr != "localhost:9080" {
		t.Fatalf("unexpected auth grpc addr: %s", cfg.Services.AuthGRPCAddr)
	}
	if cfg.Services.AuditGRPCAddr != "localhost:9082" {
		t.Fatalf("unexpected audit grpc addr: %s", cfg.Services.AuditGRPCAddr)
	}
	if cfg.Services.IAMGRPCAddr != "localhost:9081" {
		t.Fatalf("unexpected iam grpc addr: %s", cfg.Services.IAMGRPCAddr)
	}
	if cfg.Services.NotifyGRPCAddr != "localhost:9083" {
		t.Fatalf("unexpected notify grpc addr: %s", cfg.Services.NotifyGRPCAddr)
	}
}

func TestApplyEnvOverridesGRPCServiceAddresses(t *testing.T) {
	t.Setenv("OPS_AUTH_GRPC_ADDR", "auth-svc:9080")
	t.Setenv("OPS_AUDIT_GRPC_ADDR", "audit-svc:9082")
	t.Setenv("OPS_IAM_GRPC_ADDR", "iam-svc:9081")
	t.Setenv("OPS_NOTIFY_GRPC_ADDR", "notify-svc:9083")
	cfg := DefaultConfig()
	applyEnv(cfg)
	if cfg.Services.AuthGRPCAddr != "auth-svc:9080" {
		t.Fatalf("unexpected auth grpc addr: %s", cfg.Services.AuthGRPCAddr)
	}
	if cfg.Services.AuditGRPCAddr != "audit-svc:9082" {
		t.Fatalf("unexpected audit grpc addr: %s", cfg.Services.AuditGRPCAddr)
	}
	if cfg.Services.IAMGRPCAddr != "iam-svc:9081" {
		t.Fatalf("unexpected iam grpc addr: %s", cfg.Services.IAMGRPCAddr)
	}
	if cfg.Services.NotifyGRPCAddr != "notify-svc:9083" {
		t.Fatalf("unexpected notify grpc addr: %s", cfg.Services.NotifyGRPCAddr)
	}
}
