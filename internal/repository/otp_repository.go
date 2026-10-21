package repository

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisOTPRepository struct {
	client redis.Cmdable
}

var _ OTPRepository = (*redisOTPRepository)(nil)

func NewOTPRepository(client redis.Cmdable) OTPRepository {
	return &redisOTPRepository{client: client}
}

func (r *redisOTPRepository) Store(ctx context.Context, key, codeHash string, ttl time.Duration) error {
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, "otp:"+key, codeHash, ttl)
	pipe.Del(ctx, "otp:"+key+":attempts")
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisOTPRepository) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, "otp:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return value, err
}

func (r *redisOTPRepository) IncrementAttempts(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	attemptsKey := "otp:" + key + ":attempts"
	count, err := r.client.Incr(ctx, attemptsKey).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		r.client.Expire(ctx, attemptsKey, ttl)
	}
	return count, nil
}

func (r *redisOTPRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, "otp:"+key, "otp:"+key+":attempts").Err()
}

func (r *redisOTPRepository) AcquireCooldown(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, "otp:"+key+":cooldown", "1", ttl).Result()
}
