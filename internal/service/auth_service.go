package service

import (
	"context"
	"strings"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/jwt"
	"github.com/SukGebeya/Backend/pkg/password"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/google/uuid"
)

var _ Authenticator = (*AuthService)(nil)

type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtManager       *jwt.Manager
	passwordHasher   *password.Hasher
	refreshTTL       time.Duration
	otp              *OTPService
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

func (s *AuthService) WithOTP(otp *OTPService) *AuthService {
	s.otp = otp
	return s
}

// SendPhoneVerification texts a verification code to the user's phone on file.
func (s *AuthService) SendPhoneVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.Phone == nil {
		return domainerrors.BadRequest("add a phone number to your profile first")
	}
	if user.PhoneVerifiedAt != nil {
		return domainerrors.BadRequest("phone number is already verified")
	}
	return s.otp.Send(ctx, *user.Phone, OTPPurposeVerifyPhone)
}

func (s *AuthService) VerifyPhone(ctx context.Context, userID uuid.UUID, code string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Phone == nil {
		return nil, domainerrors.BadRequest("add a phone number to your profile first")
	}
	if err := s.otp.Verify(ctx, *user.Phone, OTPPurposeVerifyPhone, code); err != nil {
		return nil, err
	}
	if err := s.userRepo.MarkPhoneVerified(ctx, userID); err != nil {
		return nil, err
	}
	return s.userRepo.GetByID(ctx, userID)
}

// RequestPasswordReset texts a reset code when the phone belongs to an account.
// It reports success either way so the endpoint cannot be used to discover users.
func (s *AuthService) RequestPasswordReset(ctx context.Context, rawPhone string) error {
	normalized, err := phone.Normalize(rawPhone)
	if err != nil {
		return domainerrors.Validation("phone must be an Ethiopian mobile number")
	}
	user, err := s.userRepo.GetByPhone(ctx, normalized)
	if err != nil || !user.IsActive {
		return nil
	}
	return s.otp.Send(ctx, normalized, OTPPurposeResetPassword)
}

func (s *AuthService) ResetPassword(ctx context.Context, rawPhone, code, newPassword string) error {
	normalized, err := phone.Normalize(rawPhone)
	if err != nil {
		return domainerrors.Validation("phone must be an Ethiopian mobile number")
	}
	if err := s.otp.Verify(ctx, normalized, OTPPurposeResetPassword, code); err != nil {
		return err
	}
	user, err := s.userRepo.GetByPhone(ctx, normalized)
	if err != nil {
		return domainerrors.BadRequest("invalid code")
	}

	hash, err := s.passwordHasher.Hash(newPassword)
	if err != nil {
		return domainerrors.Internal("failed to hash password", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}
	if user.PhoneVerifiedAt == nil {
		if err := s.userRepo.MarkPhoneVerified(ctx, user.ID); err != nil {
			return err
		}
	}
	return s.refreshTokenRepo.DeleteAllForUser(ctx, user.ID)
}

type RegisterInput struct {
	Email    *string
	Phone    *string
	Password string
	Role     domain.Role
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
	if input.Email != nil && strings.TrimSpace(*input.Email) == "" {
		input.Email = nil
	}
	if input.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*input.Email))
		input.Email = &email
	}

	normalizedPhone, err := phone.NormalizePtr(input.Phone)
	if err != nil {
		return nil, domainerrors.Validation("phone must be an Ethiopian mobile number")
	}
	input.Phone = normalizedPhone

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

	role := input.Role
	if !role.IsValid() {
		role = domain.RoleBuyer
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        input.Email,
		Phone:        input.Phone,
		PasswordHash: hashedPassword,
		Role:         role,
		KYCStatus:    domain.KYCStatusPending,
		IsActive:     true,
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

	user, err := s.userRepo.GetByEmailOrPhone(ctx, NormalizeIdentifier(input.Identifier))
	if err != nil {
		return nil, domainerrors.InvalidCredentials("invalid credentials")
	}

	valid, err := s.passwordHasher.Verify(input.Password, user.PasswordHash)
	if err != nil || !valid {
		return nil, domainerrors.InvalidCredentials("invalid credentials")
	}

	if !user.IsActive {
		return nil, domainerrors.Forbidden("account is deactivated")
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
	if err == nil && !user.IsActive {
		return nil, domainerrors.Forbidden("account is deactivated")
	}
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
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, string(user.Role))
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
		ExpiresIn:    int64(s.jwtManager.AccessTTL().Seconds()),
	}, nil
}

// NormalizeIdentifier turns a login identifier into the stored form: Ethiopian
// phone numbers become +251…, anything else is treated as a lowercase email.
func NormalizeIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if normalized, err := phone.Normalize(identifier); err == nil {
		return normalized
	}
	return strings.ToLower(identifier)
}
