package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisBackend struct {
	client redis.UniversalClient
}

func NewRedisBackend(client redis.UniversalClient) *RedisBackend {
	return &RedisBackend{client: client}
}

func NewRedisBackendFromOptions(opts *redis.Options) *RedisBackend {
	return NewRedisBackend(redis.NewClient(opts))
}

func (b *RedisBackend) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := b.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return clone(value), nil
}

func (b *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return b.client.Set(ctx, key, value, ttl).Err()
}

func (b *RedisBackend) Delete(ctx context.Context, key string) error {
	return b.client.Del(ctx, key).Err()
}

func (b *RedisBackend) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	value, err := b.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if value == 1 && ttl > 0 {
		if err := b.client.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, err
		}
	}
	return value, nil
}
