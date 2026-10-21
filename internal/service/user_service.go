package service

import (
	"context"
	"strings"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/google/uuid"
)

var _ UserManager = (*UserService)(nil)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

func (s *UserService) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.User], error) {
	return s.userRepo.List(ctx, opts)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.userRepo.Delete(ctx, id)
}

type UpdateProfileInput struct {
	Email    *string
	Phone    *string
	FullName *string
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*input.Email))
		input.Email = &email
		existing, _ := s.userRepo.GetByEmail(ctx, *input.Email)
		if existing != nil && existing.ID != userID {
			return nil, domainerrors.EmailExists("email already in use")
		}
		user.Email = input.Email
	}

	phoneChanged := false
	if input.Phone != nil {
		normalized, err := phone.Normalize(*input.Phone)
		if err != nil {
			return nil, domainerrors.Validation("phone must be an Ethiopian mobile number")
		}
		existing, _ := s.userRepo.GetByPhone(ctx, normalized)
		if existing != nil && existing.ID != userID {
			return nil, domainerrors.Conflict("phone already in use")
		}
		phoneChanged = user.Phone == nil || *user.Phone != normalized
		user.Phone = &normalized
	}

	if input.FullName != nil {
		name := strings.TrimSpace(*input.FullName)
		user.FullName = &name
	}

	if input.Email == nil && input.Phone == nil && input.FullName == nil {
		return user, nil
	}

	if user.Email == nil && user.Phone == nil {
		return nil, domainerrors.Validation("email or phone is required")
	}

	updated, err := s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	if phoneChanged && updated.PhoneVerifiedAt != nil {
		if err := s.userRepo.ClearPhoneVerified(ctx, userID); err != nil {
			return nil, err
		}
		updated.PhoneVerifiedAt = nil
	}
	return updated, nil
}

func (s *UserService) UpdateKYCStatus(ctx context.Context, userID uuid.UUID, kycStatus domain.KYCStatus) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !kycStatus.IsValid() {
		return nil, domainerrors.BadRequest("invalid KYC status")
	}

	user.KYCStatus = kycStatus

	return s.userRepo.Update(ctx, user)
}
