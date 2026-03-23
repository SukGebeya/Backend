package repository

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/sony/gobreaker"
)

type CircuitBreakerRefreshTokenRepository struct {
	client    *redis.Client
	breaker   *gobreaker.CircuitBreaker
	keyPrefix string
}

func NewCircuitBreakerRefreshTokenRepository(client *redis.Client) *CircuitBreakerRefreshTokenRepository {
	settings := gobreaker.Settings{
		Name:        "Redis",
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
		},
	}

	return &CircuitBreakerRefreshTokenRepository{
		client:    client,
		breaker:   gobreaker.NewCircuitBreaker(settings),
		keyPrefix: "refresh_token:",
	}
}

func (r *CircuitBreakerRefreshTokenRepository) Store(ctx context.Context, userID uuid.UUID, tokenID string, ttl int64) error {
	_, err := r.breaker.Execute(func() (interface{}, error) {
		key := r.keyPrefix + userID.String() + ":" + tokenID
		return nil, r.client.Set(ctx, key, "1", time.Duration(ttl)*time.Second).Err()
	})
	return err
}

func (r *CircuitBreakerRefreshTokenRepository) Get(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error) {
	result, err := r.breaker.Execute(func() (interface{}, error) {
		key := r.keyPrefix + userID.String() + ":" + tokenID
		return r.client.Get(ctx, key).Result()
	})

	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return result.(string) != "", nil
}

func (r *CircuitBreakerRefreshTokenRepository) Delete(ctx context.Context, userID uuid.UUID, tokenID string) error {
	_, err := r.breaker.Execute(func() (interface{}, error) {
		key := r.keyPrefix + userID.String() + ":" + tokenID
		return nil, r.client.Del(ctx, key).Err()
	})
	return err
}

func (r *CircuitBreakerRefreshTokenRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.breaker.Execute(func() (interface{}, error) {
		pattern := r.keyPrefix + userID.String() + ":*"
		keys, err := r.client.Keys(ctx, pattern).Result()
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			return nil, nil
		}
		return nil, r.client.Del(ctx, keys...).Err()
	})
	return err
}

type CircuitBreakerRateLimitRepository struct {
	client    *redis.Client
	breaker   *gobreaker.CircuitBreaker
	keyPrefix string
}

func NewCircuitBreakerRateLimitRepository(client *redis.Client) *CircuitBreakerRateLimitRepository {
	settings := gobreaker.Settings{
		Name:        "RedisRateLimit",
		MaxRequests: 5,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
	}

	return &CircuitBreakerRateLimitRepository{
		client:    client,
		breaker:   gobreaker.NewCircuitBreaker(settings),
		keyPrefix: "rate_limit:",
	}
}

func (r *CircuitBreakerRateLimitRepository) Increment(ctx context.Context, key string, window int64) (int64, error) {
	result, err := r.breaker.Execute(func() (interface{}, error) {
		fullKey := r.keyPrefix + key
		val, err := r.client.Incr(ctx, fullKey).Result()
		if err != nil {
			return nil, err
		}
		if val == 1 {
			if err := r.client.Expire(ctx, fullKey, time.Duration(window)*time.Second).Err(); err != nil {
				return nil, err
			}
		}
		return val, nil
	})

	if err != nil {
		return 0, err
	}

	return result.(int64), nil
}

func (r *CircuitBreakerRateLimitRepository) GetTTL(ctx context.Context, key string) (int64, error) {
	result, err := r.breaker.Execute(func() (interface{}, error) {
		fullKey := r.keyPrefix + key
		return r.client.TTL(ctx, fullKey).Result()
	})

	if err != nil {
		return 0, err
	}

	return int64(result.(time.Duration).Seconds()), nil
}

func (r *CircuitBreakerRateLimitRepository) Reset(ctx context.Context, key string) error {
	_, err := r.breaker.Execute(func() (interface{}, error) {
		fullKey := r.keyPrefix + key
		return nil, r.client.Del(ctx, fullKey).Err()
	})
	return err
}
