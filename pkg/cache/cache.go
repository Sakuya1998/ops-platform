package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

type Backend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type CounterBackend interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type Loader func(ctx context.Context) ([]byte, time.Duration, error)

type Options struct {
	DefaultTTL     time.Duration
	MaxEntries     int
	L2             Backend
	IgnoreL2Errors bool
	Now            func() time.Time
}

type Cache struct {
	mu         sync.RWMutex
	items      map[string]entry
	defaultTTL time.Duration
	maxEntries int
	l2         Backend
	ignoreL2   bool
	now        func() time.Time
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

func New(opts Options) *Cache {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	if opts.DefaultTTL <= 0 {
		opts.DefaultTTL = 5 * time.Minute
	}
	return &Cache{
		items:      make(map[string]entry),
		defaultTTL: opts.DefaultTTL,
		maxEntries: opts.MaxEntries,
		l2:         opts.L2,
		ignoreL2:   opts.IgnoreL2Errors,
		now:        now,
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if value, ok := c.getL1(key); ok {
		return value, nil
	}
	if c.l2 == nil {
		return nil, ErrNotFound
	}
	value, err := c.l2.Get(ctx, key)
	if err != nil {
		if c.ignoreL2 {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.setL1(key, value, c.defaultTTL)
	return clone(value), nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}
	c.setL1(key, value, ttl)
	if c.l2 != nil {
		if err := c.l2.Set(ctx, key, clone(value), ttl); err != nil && !c.ignoreL2 {
			return err
		}
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	if c.l2 != nil {
		if err := c.l2.Delete(ctx, key); err != nil && !c.ignoreL2 {
			return err
		}
	}
	return nil
}

func (c *Cache) GetOrLoad(ctx context.Context, key string, loader Loader) ([]byte, error) {
	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	value, ttl, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.Set(ctx, key, value, ttl); err != nil {
		return nil, err
	}
	return clone(value), nil
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *Cache) getL1(key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !item.expiresAt.IsZero() && c.now().After(item.expiresAt) {
		c.mu.Lock()
		if current, ok := c.items[key]; ok && current.expiresAt.Equal(item.expiresAt) {
			delete(c.items, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	return clone(item.value), true
}

func (c *Cache) setL1(key string, value []byte, ttl time.Duration) {
	expiresAt := c.now().Add(ttl)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{value: clone(value), expiresAt: expiresAt}
	if c.maxEntries > 0 && len(c.items) > c.maxEntries {
		c.evictOneLocked()
	}
}

func (c *Cache) evictOneLocked() {
	var oldestKey string
	var oldest time.Time
	for key, item := range c.items {
		if oldestKey == "" || item.expiresAt.Before(oldest) {
			oldestKey = key
			oldest = item.expiresAt
		}
	}
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

func clone(value []byte) []byte {
	if value == nil {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}
