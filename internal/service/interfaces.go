package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/SukGebeya/Backend/internal/domain"
)

type Authenticator interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
}

type UserManager interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error)
}

type MerchantOnboarder interface {
	Onboard(ctx context.Context, input OnboardInput) (*domain.MerchantProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error)
	GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error)
	Update(ctx context.Context, merchantID uuid.UUID, input UpdateMerchantInput) (*domain.MerchantProfile, error)
}
