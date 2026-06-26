package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCacheSetGetCopiesValue(t *testing.T) {
	ctx := context.Background()
	c := New(Options{DefaultTTL: time.Minute})
	value := []byte("hello")
	if err := c.Set(ctx, "k", value, time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	value[0] = 'x'
	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
	got[0] = 'x'
	got, _ = c.Get(ctx, "k")
	if string(got) != "hello" {
		t.Fatalf("cache returned mutable backing slice")
	}
}

func TestCacheTTL(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := New(Options{DefaultTTL: time.Minute, Now: func() time.Time { return now }})
	_ = c.Set(ctx, "k", []byte("v"), time.Second)
	now = now.Add(2 * time.Second)
	_, err := c.Get(ctx, "k")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCacheL2Backfill(t *testing.T) {
	ctx := context.Background()
	l2 := NewMemoryBackend(time.Minute)
	_ = l2.Set(ctx, "k", []byte("from-l2"), time.Minute)
	c := New(Options{DefaultTTL: time.Minute, L2: l2})
	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "from-l2" {
		t.Fatalf("expected from-l2, got %s", got)
	}
	if c.Len() != 1 {
		t.Fatalf("expected L1 backfill, len=%d", c.Len())
	}
}

func TestCacheGetOrLoad(t *testing.T) {
	ctx := context.Background()
	calls := 0
	c := New(Options{DefaultTTL: time.Minute})
	loader := func(ctx context.Context) ([]byte, time.Duration, error) {
		calls++
		return []byte("loaded"), time.Minute, nil
	}
	for i := 0; i < 2; i++ {
		got, err := c.GetOrLoad(ctx, "k", loader)
		if err != nil {
			t.Fatalf("GetOrLoad: %v", err)
		}
		if string(got) != "loaded" {
			t.Fatalf("expected loaded, got %s", got)
		}
	}
	if calls != 1 {
		t.Fatalf("expected loader once, got %d", calls)
	}
}

func TestCacheMaxEntries(t *testing.T) {
	ctx := context.Background()
	c := New(Options{DefaultTTL: time.Minute, MaxEntries: 1})
	_ = c.Set(ctx, "a", []byte("a"), time.Minute)
	_ = c.Set(ctx, "b", []byte("b"), time.Minute)
	if c.Len() != 1 {
		t.Fatalf("expected len 1, got %d", c.Len())
	}
}

type failingBackend struct{}

func (failingBackend) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, errors.New("l2 down")
}

func (failingBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return errors.New("l2 down")
}

func (failingBackend) Delete(ctx context.Context, key string) error {
	return errors.New("l2 down")
}

func TestCacheIgnoreL2Errors(t *testing.T) {
	ctx := context.Background()
	c := New(Options{DefaultTTL: time.Minute, L2: failingBackend{}, IgnoreL2Errors: true})
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set should ignore L2 error: %v", err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get should hit L1: %v", err)
	}
	if string(got) != "v" {
		t.Fatalf("expected v, got %s", got)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete should ignore L2 error: %v", err)
	}
}

func TestMemoryBackendIncrementUsesTTL(t *testing.T) {
	ctx := context.Background()
	backend := NewMemoryBackend(50 * time.Millisecond)

	first, err := backend.Increment(ctx, "login:attempt", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Increment first: %v", err)
	}
	second, err := backend.Increment(ctx, "login:attempt", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Increment second: %v", err)
	}
	if first != 1 || second != 2 {
		t.Fatalf("expected increments 1 then 2, got %d then %d", first, second)
	}

	time.Sleep(70 * time.Millisecond)
	afterTTL, err := backend.Increment(ctx, "login:attempt", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Increment after ttl: %v", err)
	}
	if afterTTL != 1 {
		t.Fatalf("expected ttl reset to 1, got %d", afterTTL)
	}
}
