package cache

import (
	"context"
	"strconv"
	"time"
)

type MemoryBackend struct {
	cache *Cache
}

func NewMemoryBackend(defaultTTL time.Duration) *MemoryBackend {
	return &MemoryBackend{cache: New(Options{DefaultTTL: defaultTTL})}
}

func (b *MemoryBackend) Get(ctx context.Context, key string) ([]byte, error) {
	return b.cache.Get(ctx, key)
}

func (b *MemoryBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return b.cache.Set(ctx, key, value, ttl)
}

func (b *MemoryBackend) Delete(ctx context.Context, key string) error {
	return b.cache.Delete(ctx, key)
}

func (b *MemoryBackend) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	b.cache.mu.Lock()
	defer b.cache.mu.Unlock()

	now := b.cache.now()
	current := int64(0)
	if item, ok := b.cache.items[key]; ok {
		if item.expiresAt.IsZero() || now.Before(item.expiresAt) {
			if parsed, err := strconv.ParseInt(string(item.value), 10, 64); err == nil {
				current = parsed
			}
		}
	}
	current++
	if ttl <= 0 {
		ttl = b.cache.defaultTTL
	}
	b.cache.items[key] = entry{
		value:     []byte(strconv.FormatInt(current, 10)),
		expiresAt: now.Add(ttl),
	}
	if b.cache.maxEntries > 0 && len(b.cache.items) > b.cache.maxEntries {
		b.cache.evictOneLocked()
	}
	return current, nil
}
