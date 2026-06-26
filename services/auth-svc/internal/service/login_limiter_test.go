package service

import (
	"context"
	"testing"
	"time"

	"github.com/Sakuya1998/ops-platform/pkg/cache"
)

func TestLoginLimiterBlocksAfterLimit(t *testing.T) {
	limiter := NewLoginLimiter(cache.NewMemoryBackend(time.Minute), LoginLimiterOptions{
		MaxAttempts: 2,
		Window:      time.Minute,
	})
	ctx := context.Background()
	attempt := LoginAttemptKey{
		OrgID:    "org-1",
		Provider: "local",
		Username: "admin",
		IP:       "127.0.0.1",
	}

	if err := limiter.Allow(ctx, attempt); err != nil {
		t.Fatalf("first Allow: %v", err)
	}
	limiter.RecordFailure(ctx, attempt)
	if err := limiter.Allow(ctx, attempt); err != nil {
		t.Fatalf("second Allow: %v", err)
	}
	limiter.RecordFailure(ctx, attempt)
	if err := limiter.Allow(ctx, attempt); err == nil {
		t.Fatal("expected third attempt to be blocked")
	}
}

func TestLoginLimiterRecordSuccessClearsFailureCounters(t *testing.T) {
	limiter := NewLoginLimiter(cache.NewMemoryBackend(time.Minute), LoginLimiterOptions{
		MaxAttempts: 1,
		Window:      time.Minute,
	})
	ctx := context.Background()
	attempt := LoginAttemptKey{
		OrgID:    "org-1",
		Provider: "local",
		Username: "admin",
		IP:       "127.0.0.1",
	}

	limiter.RecordFailure(ctx, attempt)
	if err := limiter.Allow(ctx, attempt); err == nil {
		t.Fatal("expected attempt to be blocked before success reset")
	}
	limiter.RecordSuccess(ctx, attempt)
	if err := limiter.Allow(ctx, attempt); err != nil {
		t.Fatalf("expected success reset to allow login, got %v", err)
	}
}
