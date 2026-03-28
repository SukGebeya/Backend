package repository

import (
	"context"
	"errors"

	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type merchantRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewMerchantRepository(pool *pgxpool.Pool) MerchantRepository {
	return &merchantRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

var _ MerchantRepository = (*merchantRepository)(nil)

func (r *merchantRepository) WithTx(tx pgx.Tx) MerchantRepository {
	return &merchantRepository{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *merchantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MerchantProfile, error) {
	merchant, err := r.queries.GetMerchantProfileByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant", err)
	}
	return dbMerchantToDomain(merchant), nil
}

func (r *merchantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error) {
	merchant, err := r.queries.GetMerchantProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant by user ID", err)
	}
	return dbMerchantToDomain(merchant), nil
}

func (r *merchantRepository) GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error) {
	merchant, err := r.queries.GetMerchantProfileBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant by slug", err)
	}
	return dbMerchantToDomain(merchant), nil
}

func (r *merchantRepository) Create(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error) {
	created, err := r.queries.CreateMerchantProfile(ctx, db.CreateMerchantProfileParams{
		ID:              merchant.ID,
		UserID:          merchant.UserID,
		ShopName:        merchant.ShopName,
		ShopSlug:        merchant.ShopSlug,
		ShopDescription: merchant.ShopDescription,
		ShopLogoUrl:     merchant.ShopLogoURL,
		ShopBannerUrl:   merchant.ShopBannerURL,
		ContactPhone:    merchant.ContactPhone,
		ContactEmail:    merchant.ContactEmail,
		Address:         merchant.Address,
		City:            merchant.City,
		BlnkBalanceID:   merchant.BlnkBalanceID,
		BlnkLedgerID:    merchant.BlnkLedgerID,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create merchant profile", err)
	}
	return dbMerchantToDomain(created), nil
}

func (r *merchantRepository) Update(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error) {
	updated, err := r.queries.UpdateMerchantProfile(ctx, db.UpdateMerchantProfileParams{
		ID:              merchant.ID,
		ShopName:        merchant.ShopName,
		ShopDescription: merchant.ShopDescription,
		ShopLogoUrl:     merchant.ShopLogoURL,
		ShopBannerUrl:   merchant.ShopBannerURL,
		ContactPhone:    merchant.ContactPhone,
		ContactEmail:    merchant.ContactEmail,
		Address:         merchant.Address,
		City:            merchant.City,
		BlnkBalanceID:   merchant.BlnkBalanceID,
		BlnkLedgerID:    merchant.BlnkLedgerID,
		IsVerified:      merchant.IsVerified,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to update merchant profile", err)
	}
	return dbMerchantToDomain(updated), nil
}

func (r *merchantRepository) UpdateBlnkWallet(ctx context.Context, id uuid.UUID, balanceID, ledgerID string) (*domain.MerchantProfile, error) {
	updated, err := r.queries.UpdateMerchantBlnkWallet(ctx, db.UpdateMerchantBlnkWalletParams{
		ID:            id,
		BlnkBalanceID: &balanceID,
		BlnkLedgerID:  &ledgerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to update merchant blnk wallet", err)
	}
	return dbMerchantToDomain(updated), nil
}

func (r *merchantRepository) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.MerchantProfile], error) {
	merchants, err := r.queries.ListMerchantProfiles(ctx, db.ListMerchantProfilesParams{
		PageLimit:  int32(opts.Limit()),
		PageOffset: int32(opts.Offset()),
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list merchants", err)
	}

	items := make([]domain.MerchantProfile, len(merchants))
	for i, m := range merchants {
		items[i] = *dbMerchantToDomain(m)
	}

	return &domain.PaginatedResult[domain.MerchantProfile]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func dbMerchantToDomain(m db.MerchantProfile) *domain.MerchantProfile {
	return &domain.MerchantProfile{
		ID:              m.ID,
		UserID:          m.UserID,
		ShopName:        m.ShopName,
		ShopSlug:        m.ShopSlug,
		ShopDescription: m.ShopDescription,
		ShopLogoURL:     m.ShopLogoUrl,
		ShopBannerURL:   m.ShopBannerUrl,
		ContactPhone:    m.ContactPhone,
		ContactEmail:    m.ContactEmail,
		Address:         m.Address,
		City:            m.City,
		BlnkBalanceID:   m.BlnkBalanceID,
		BlnkLedgerID:    m.BlnkLedgerID,
		IsVerified:      m.IsVerified,
		TotalSalesCents: m.TotalSalesCents,
		CreatedAt:       m.CreatedAt.Time,
		UpdatedAt:       m.UpdatedAt.Time,
	}
}
