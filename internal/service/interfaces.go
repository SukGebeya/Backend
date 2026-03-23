package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
)

type Authenticator interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
}

type NoteManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Note, error)
	GetByIDAndUserID(ctx context.Context, id, userID uuid.UUID) (*domain.Note, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, opts domain.NoteListOptions) (*domain.PaginatedResult[domain.Note], error)
	Create(ctx context.Context, input CreateNoteInput) (*domain.Note, error)
	Update(ctx context.Context, input UpdateNoteInput) (*domain.Note, error)
	SoftDelete(ctx context.Context, id, userID uuid.UUID) error
	Restore(ctx context.Context, id, userID uuid.UUID) error
}

type UserManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error)
}
