package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	GetByEmailOrPhone(ctx context.Context, identifier string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
	Count(ctx context.Context) (int64, error)
}

type NoteRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error)
	GetByIDWithDeleted(ctx context.Context, id uuid.UUID) (*domain.Note, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, opts domain.NoteListOptions) (*domain.PaginatedResult[domain.Note], error)
	Create(ctx context.Context, note *domain.Note) (*domain.Note, error)
	Update(ctx context.Context, note *domain.Note) (*domain.Note, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) error
	Restore(ctx context.Context, id, userID uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}

type RefreshTokenRepository interface {
	Store(ctx context.Context, userID uuid.UUID, tokenID string, ttl int64) error
	Get(ctx context.Context, userID uuid.UUID, tokenID string) (bool, error)
	Delete(ctx context.Context, userID uuid.UUID, tokenID string) error
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error
}

type RateLimitRepository interface {
	Increment(ctx context.Context, key string, window int64) (int64, error)
	GetTTL(ctx context.Context, key string) (int64, error)
	Reset(ctx context.Context, key string) error
}
