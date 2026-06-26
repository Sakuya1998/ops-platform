package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ops-platform/pkg/cache"
)

const (
	defaultLoginLimiterMaxAttempts = 10
	defaultLoginLimiterWindow      = 15 * time.Minute
)

type LoginLimiterOptions struct {
	MaxAttempts int
	Window      time.Duration
}

type LoginAttemptKey struct {
	OrgID    string
	Provider string
	Username string
	IP       string
}

type LoginLimiter struct {
	store       loginAttemptStore
	maxAttempts int
	window      time.Duration
}

type loginAttemptStore interface {
	cache.CounterBackend
	Get(context.Context, string) ([]byte, error)
	Delete(context.Context, string) error
}

func NewLoginLimiter(store loginAttemptStore, opts LoginLimiterOptions) *LoginLimiter {
	if store == nil {
		return nil
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = defaultLoginLimiterMaxAttempts
	}
	if opts.Window <= 0 {
		opts.Window = defaultLoginLimiterWindow
	}
	return &LoginLimiter{store: store, maxAttempts: opts.MaxAttempts, window: opts.Window}
}

func (l *LoginLimiter) Allow(ctx context.Context, attempt LoginAttemptKey) error {
	if l == nil {
		return nil
	}
	count, err := l.failureCount(ctx, attempt)
	if err != nil {
		return err
	}
	if count >= int64(l.maxAttempts) {
		return fmt.Errorf("too many login attempts, please try again later")
	}
	return nil
}

func (l *LoginLimiter) RecordFailure(ctx context.Context, attempt LoginAttemptKey) {
	if l == nil {
		return
	}
	_, _ = l.store.Increment(ctx, l.failureKey(attempt), l.window)
}

func (l *LoginLimiter) RecordSuccess(ctx context.Context, attempt LoginAttemptKey) {
	if l == nil {
		return
	}
	_ = l.store.Delete(ctx, l.failureKey(attempt))
}

func (l *LoginLimiter) failureKey(attempt LoginAttemptKey) string {
	return "auth:login:failure:" + attempt.fingerprint()
}

func (l *LoginLimiter) failureCount(ctx context.Context, attempt LoginAttemptKey) (int64, error) {
	value, err := l.store.Get(ctx, l.failureKey(attempt))
	if err != nil {
		if err == cache.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	count, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

func (k LoginAttemptKey) fingerprint() string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(k.OrgID)),
		strings.ToLower(strings.TrimSpace(k.Provider)),
		strings.ToLower(strings.TrimSpace(k.Username)),
		strings.TrimSpace(k.IP),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
