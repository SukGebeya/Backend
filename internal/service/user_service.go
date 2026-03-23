package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nate/go-boilerplate/internal/domain"
	domainerrors "github.com/nate/go-boilerplate/internal/errors"
	"github.com/nate/go-boilerplate/internal/repository"
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
	Email *string
	Phone *string
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.Email != nil {
		existing, _ := s.userRepo.GetByEmail(ctx, *input.Email)
		if existing != nil && existing.ID != userID {
			return nil, domainerrors.EmailExists("email already in use")
		}
		user.Email = input.Email
	}

	if input.Phone != nil {
		existing, _ := s.userRepo.GetByPhone(ctx, *input.Phone)
		if existing != nil && existing.ID != userID {
			return nil, domainerrors.Conflict("phone already in use")
		}
		user.Phone = input.Phone
	}

	if input.Email == nil && input.Phone == nil {
		return user, nil
	}

	if user.Email == nil && user.Phone == nil {
		return nil, domainerrors.Validation("email or phone is required")
	}

	return s.userRepo.Update(ctx, user)
}
