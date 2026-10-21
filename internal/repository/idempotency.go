package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyRepository struct {
	client redis.Cmdable
}

func NewIdempotencyRepository(client redis.Cmdable) IdempotencyRepository {
	return &RedisIdempotencyRepository{client: client}
}

func (r *RedisIdempotencyRepository) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return result, nil
}

func (r *RedisIdempotencyRepository) Set(ctx context.Context, key string, value string, ttl int64) error {
	return r.client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
}

func (r *RedisIdempotencyRepository) Lock(ctx context.Context, key string, ttl int64) (bool, error) {
	return r.client.SetNX(ctx, key, "1", time.Duration(ttl)*time.Second).Result()
}

func (r *RedisIdempotencyRepository) Unlock(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
