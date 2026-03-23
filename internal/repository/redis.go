package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type refreshTokenRepository struct {
	client *redis.Client
}

func NewRefreshTokenRepository(client *redis.Client) RefreshTokenRepository {
	return &refreshTokenRepository{client: client}
}

func (r *refreshTokenRepository) key(userID uuid.UUID, tokenID string) string {
	return fmt.Sprintf("refresh_token:%s:%s", userID, tokenID)
}

func (r *refreshTokenRepository) Store(ctx context.Context, userID uuid.UUID, tokenID string, ttl int64) error {
	key := r.key(userID, tokenID)
	return r.client.Set(ctx, key, "1", time.Duration(ttl)*time.Second).Err()
}

func (r *refreshTokenRepository) Get(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error) {
	key := r.key(userID, tokenID)
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result == "1", nil
}

func (r *refreshTokenRepository) Delete(ctx context.Context, userID uuid.UUID, tokenID string) error {
	key := r.key(userID, tokenID)
	return r.client.Del(ctx, key).Err()
}

func (r *refreshTokenRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	pattern := fmt.Sprintf("refresh_token:%s:*", userID)
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

type rateLimitRepository struct {
	client *redis.Client
}

func NewRateLimitRepository(client *redis.Client) RateLimitRepository {
	return &rateLimitRepository{client: client}
}

func (r *rateLimitRepository) key(identifier string) string {
	return fmt.Sprintf("rate_limit:%s", identifier)
}

func (r *rateLimitRepository) Increment(ctx context.Context, identifier string, window int64) (int64, error) {
	key := r.key(identifier)
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		r.client.Expire(ctx, key, time.Duration(window)*time.Second)
	}

	return count, nil
}

func (r *rateLimitRepository) GetTTL(ctx context.Context, identifier string) (int64, error) {
	key := r.key(identifier)
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int64(ttl.Seconds()), nil
}

func (r *rateLimitRepository) Reset(ctx context.Context, identifier string) error {
	key := r.key(identifier)
	return r.client.Del(ctx, key).Err()
}
