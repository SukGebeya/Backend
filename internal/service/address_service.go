package service

import (
	"context"

	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/google/uuid"
)

var _ AddressManager = (*AddressService)(nil)

type AddressService struct {
	repo repository.AddressRepository
}

func NewAddressService(repo repository.AddressRepository) *AddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Address, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AddressService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *AddressService) GetDefault(ctx context.Context, userID uuid.UUID) (*domain.Address, error) {
	return s.repo.GetDefault(ctx, userID)
}

func (s *AddressService) Create(ctx context.Context, input CreateAddressInput) (*domain.Address, error) {
	count, err := s.repo.CountByUserID(ctx, input.UserID)
	if err != nil {
		return nil, domainerrors.Internal("failed to count addresses", err)
	}

	if count >= 10 {
		return nil, domainerrors.BadRequest("maximum of 10 addresses allowed per user")
	}

	if input.IsDefault {
		if err := s.repo.ClearDefaults(ctx, input.UserID); err != nil {
			return nil, domainerrors.Internal("failed to clear default address", err)
		}
	}

	address := &domain.Address{
		ID:        uuid.New(),
		UserID:    input.UserID,
		FullName:  input.FullName,
		Phone:     input.Phone,
		Line1:     input.Line1,
		Line2:     input.Line2,
		City:      input.City,
		Subcity:   input.Subcity,
		Woreda:    input.Woreda,
		IsDefault: input.IsDefault,
	}

	created, err := s.repo.Create(ctx, address)
	if err != nil {
		return nil, domainerrors.Internal("failed to create address", err)
	}

	return created, nil
}

func (s *AddressService) Update(ctx context.Context, id uuid.UUID, input UpdateAddressInput) (*domain.Address, error) {
	address, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.FullName != nil {
		address.FullName = *input.FullName
	}
	if input.Phone != nil {
		address.Phone = *input.Phone
	}
	if input.Line1 != nil {
		address.Line1 = *input.Line1
	}
	if input.Line2 != nil {
		address.Line2 = input.Line2
	}
	if input.City != nil {
		address.City = *input.City
	}
	if input.Subcity != nil {
		address.Subcity = input.Subcity
	}
	if input.Woreda != nil {
		address.Woreda = input.Woreda
	}

	return s.repo.Update(ctx, address)
}

func (s *AddressService) SetDefault(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error) {
	return s.repo.SetDefault(ctx, userID, addressID)
}

func (s *AddressService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
