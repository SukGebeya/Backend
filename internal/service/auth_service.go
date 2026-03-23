package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
	domainerrors "github.com/nate/go-boilerplate/internal/errors"
	"github.com/nate/go-boilerplate/internal/repository"
	"github.com/nate/go-boilerplate/pkg/jwt"
	"github.com/nate/go-boilerplate/pkg/password"
)

var _ Authenticator = (*AuthService)(nil)

type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *jwt.Manager
	passwordHasher   *password.Hasher
	refreshTTL       time.Duration
}

func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtManager *jwt.Manager,
	passwordHasher *password.Hasher,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtManager:       jwtManager,
		passwordHasher:   passwordHasher,
		refreshTTL:       refreshTTL,
	}
}

type RegisterInput struct {
	Email    *string
	Phone    *string
	Password string
}

type LoginInput struct {
	Identifier string
	Password   string
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type AuthResult struct {
	User   *domain.User
	Tokens *AuthTokens
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	if input.Email == nil && input.Phone == nil {
		return nil, domainerrors.Validation("email or phone is required")
	}

	if input.Email != nil {
		existing, _ := s.userRepo.GetByEmail(ctx, *input.Email)
		if existing != nil {
			return nil, domainerrors.EmailExists("email already registered")
		}
	}

	if input.Phone != nil {
		existing, _ := s.userRepo.GetByPhone(ctx, *input.Phone)
		if existing != nil {
			return nil, domainerrors.Conflict("phone already registered")
		}
	}

	hashedPassword, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, domainerrors.Internal("failed to hash password", err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: hashedPassword,
		Role:         domain.RoleUser,
	}

	created, err := s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	tokens, err := s.generateTokens(ctx, created)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:   created,
		Tokens: tokens,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	if input.Identifier == "" {
		return nil, domainerrors.Validation("identifier is required")
	}

	user, err := s.userRepo.GetByEmailOrPhone(ctx, input.Identifier)
	if err != nil {
		return nil, domainerrors.InvalidCredentials("invalid credentials")
	}

	valid, err := s.passwordHasher.Verify(input.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, domainerrors.InvalidCredentials("invalid credentials")
	}

	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:   user,
		Tokens: tokens,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	userID, tokenID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, domainerrors.InvalidToken("invalid refresh token")
	}

	valid, err := s.refreshTokenRepo.Get(ctx, userID, tokenID)
	if err != nil || !valid {
		return nil, domainerrors.InvalidToken("refresh token not found or revoked")
	}

	if err := s.refreshTokenRepo.Delete(ctx, userID, tokenID); err != nil {
		return nil, domainerrors.Internal("failed to revoke old refresh token", err)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, domainerrors.UserNotFound("user not found")
	}

	return s.generateTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	userID, tokenID, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return domainerrors.InvalidToken("invalid refresh token")
	}

	return s.refreshTokenRepo.Delete(ctx, userID, tokenID)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.refreshTokenRepo.DeleteAllForUser(ctx, userID)
}

func (s *AuthService) generateTokens(ctx context.Context, user *domain.User) (*AuthTokens, error) {
	identifier := user.DisplayName()

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, identifier, string(user.Role))
	if err != nil {
		return nil, domainerrors.Internal("failed to generate access token", err)
	}

	refreshToken, tokenID, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, domainerrors.Internal("failed to generate refresh token", err)
	}

	if err := s.refreshTokenRepo.Store(ctx, user.ID, tokenID, int64(s.refreshTTL.Seconds())); err != nil {
		return nil, domainerrors.Internal("failed to store refresh token", err)
	}

	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.refreshTTL.Seconds()),
	}, nil
}
