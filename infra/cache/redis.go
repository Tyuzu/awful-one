package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

/* -------------------- Basic KV -------------------- */

func (r *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return []byte{}, nil
	}
	return val, err
}

func (r *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	ok, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (r *RedisCache) SetWithExpiry(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.SetEx(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *RedisCache) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

/* -------------------- Hash -------------------- */

func (r *RedisCache) HSet(ctx context.Context, key, field string, value []byte) error {
	return r.client.HSet(ctx, key, field, value).Err()
}

func (r *RedisCache) HGet(ctx context.Context, key, field string) ([]byte, error) {
	val, err := r.client.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return []byte{}, nil
	}
	return val, err
}

func (r *RedisCache) HDel(ctx context.Context, key, field string) (bool, error) {
	n, err := r.client.HDel(ctx, key, field).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

/* -------------------- Counters -------------------- */

func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

/* -------------------- Pub/Sub -------------------- */

func (r *RedisCache) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return r.client.Subscribe(ctx, channel)
}

func (r *RedisCache) Publish(ctx context.Context, channel string, message any) error {
	return r.client.Publish(ctx, channel, message).Err()
}

/* -------------------- Helpers -------------------- */

func IsRedisNil(err error) bool {
	return err == redis.Nil
}
