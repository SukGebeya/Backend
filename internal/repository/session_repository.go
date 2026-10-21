package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type sessionRepository struct {
	client redis.Cmdable
}

func NewSessionRepository(client redis.Cmdable) SessionRepository {
	return &sessionRepository{client: client}
}

var _ SessionRepository = (*sessionRepository)(nil)

const sessionKeyPrefix = "session:"

func (r *sessionRepository) Create(ctx context.Context, sessionID string, ttl int64) error {
	key := sessionKeyPrefix + sessionID
	return r.client.Set(ctx, key, "1", time.Duration(ttl)*time.Second).Err()
}

func (r *sessionRepository) Get(ctx context.Context, sessionID string) (bool, error) {
	key := sessionKeyPrefix + sessionID
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	return val == "1", nil
}

func (r *sessionRepository) Delete(ctx context.Context, sessionID string) error {
	key := sessionKeyPrefix + sessionID
	return r.client.Del(ctx, key).Err()
}

func (r *sessionRepository) AssociateUser(ctx context.Context, sessionID string, userID uuid.UUID) error {
	key := sessionKeyPrefix + sessionID
	return r.client.Set(ctx, key, userID.String(), 7*24*time.Hour).Err()
}
