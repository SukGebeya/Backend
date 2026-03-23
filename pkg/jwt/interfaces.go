package jwt

import (
	"context"

	"github.com/google/uuid"
)

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email, role string) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, string, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (uuid.UUID, string, error)
}

type TokenStorage interface {
	Store(ctx context.Context, userID uuid.UUID, tokenID string, ttl int64) error
	Get(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID uuid.UUID, tokenID string) error
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}

var _ TokenManager = (*Manager)(nil)
