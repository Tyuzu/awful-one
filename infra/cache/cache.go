package cache

import (
	"context"
	"time"
)

type Cache interface {
	/* KV */
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error)
	SetWithExpiry(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Exists(ctx context.Context, key string) (bool, error)
	Del(ctx context.Context, key string) error

	/* Hash */
	HSet(ctx context.Context, key, field string, value []byte) error
	HGet(ctx context.Context, key, field string) ([]byte, error)
	HDel(ctx context.Context, key, field string) (bool, error)

	/* Counters */
	Incr(ctx context.Context, key string) (int64, error)
}
