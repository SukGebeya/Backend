package repository

import (
	"context"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type addressRepository struct {
	queries *db.Queries
}

func NewAddressRepository(pool *pgxpool.Pool) AddressRepository {
	return &addressRepository{queries: db.New(pool)}
}

var _ AddressRepository = (*addressRepository)(nil)

func (r *addressRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Address, error) {
	addr, err := r.queries.GetAddressByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("address not found")
		}
		return nil, domainerrors.Internal("failed to get address", err)
	}
	return dbAddressToDomain(addr), nil
}

func (r *addressRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Address, error) {
	addrs, err := r.queries.ListAddressesByUserID(ctx, userID)
	if err != nil {
		return nil, domainerrors.Internal("failed to list addresses", err)
	}
	result := make([]domain.Address, len(addrs))
	for i, a := range addrs {
		result[i] = *dbAddressToDomain(a)
	}
	return result, nil
}

func (r *addressRepository) GetDefault(ctx context.Context, userID uuid.UUID) (*domain.Address, error) {
	addr, err := r.queries.GetDefaultAddress(ctx, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("default address not found")
		}
		return nil, domainerrors.Internal("failed to get default address", err)
	}
	return dbAddressToDomain(addr), nil
}

func (r *addressRepository) Create(ctx context.Context, address *domain.Address) (*domain.Address, error) {
	addr, err := r.queries.CreateAddress(ctx, db.CreateAddressParams{
		ID:        address.ID,
		UserID:    address.UserID,
		FullName:  address.FullName,
		Phone:     address.Phone,
		Line1:     address.Line1,
		Line2:     address.Line2,
		City:      address.City,
		Subcity:   address.Subcity,
		Woreda:    address.Woreda,
		IsDefault: address.IsDefault,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create address", err)
	}
	return dbAddressToDomain(addr), nil
}

func (r *addressRepository) Update(ctx context.Context, address *domain.Address) (*domain.Address, error) {
	addr, err := r.queries.UpdateAddress(ctx, db.UpdateAddressParams{
		ID:        address.ID,
		FullName:  address.FullName,
		Phone:     address.Phone,
		Line1:     address.Line1,
		Line2:     address.Line2,
		City:      address.City,
		Subcity:   address.Subcity,
		Woreda:    address.Woreda,
		IsDefault: address.IsDefault,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("address not found")
		}
		return nil, domainerrors.Internal("failed to update address", err)
	}
	return dbAddressToDomain(addr), nil
}

func (r *addressRepository) SetDefault(ctx context.Context, userID, addressID uuid.UUID) (*domain.Address, error) {
	if err := r.queries.SetDefaultAddress(ctx, userID); err != nil {
		return nil, domainerrors.Internal("failed to clear default addresses", err)
	}
	addr, err := r.queries.SetAddressAsDefault(ctx, addressID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domainerrors.NotFound("address not found")
		}
		return nil, domainerrors.Internal("failed to set default address", err)
	}
	return dbAddressToDomain(addr), nil
}

func (r *addressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteAddress(ctx, id); err != nil {
		return domainerrors.Internal("failed to delete address", err)
	}
	return nil
}

func (r *addressRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := r.queries.CountAddressesByUser(ctx, userID)
	if err != nil {
		return 0, domainerrors.Internal("failed to count addresses", err)
	}
	return count, nil
}

func dbAddressToDomain(a db.Address) *domain.Address {
	return &domain.Address{
		ID:        a.ID,
		UserID:    a.UserID,
		FullName:  a.FullName,
		Phone:     a.Phone,
		Line1:     a.Line1,
		Line2:     a.Line2,
		City:      a.City,
		Subcity:   a.Subcity,
		Woreda:    a.Woreda,
		IsDefault: a.IsDefault,
		CreatedAt: a.CreatedAt.Time,
	}
}
