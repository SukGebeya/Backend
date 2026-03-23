package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/SukGebeya/Backend/internal/db"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
)

type merchantRepository struct {
	queries *db.Queries
}

func NewMerchantRepository(pool *pgxpool.Pool) MerchantRepository {
	return &merchantRepository{queries: db.New(pool)}
}

var _ MerchantRepository = (*merchantRepository)(nil)

func (r *merchantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.MerchantProfile, error) {
	var pgID pgtype.UUID
	_ = pgID.Scan(id.String())

	merchant, err := r.queries.GetMerchantProfileByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant", err)
	}
	return dbMerchantRowToDomain(&merchant), nil
}

func (r *merchantRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error) {
	var pgID pgtype.UUID
	_ = pgID.Scan(userID.String())

	merchant, err := r.queries.GetMerchantProfileByUserID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant by user ID", err)
	}
	return dbMerchantRowToDomain(&merchant), nil
}

func (r *merchantRepository) GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error) {
	merchant, err := r.queries.GetMerchantProfileBySlug(ctx, &slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to get merchant by slug", err)
	}
	return dbMerchantRowToDomain(&merchant), nil
}

func (r *merchantRepository) Create(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error) {
	var pgID, pgUserID pgtype.UUID
	_ = pgID.Scan(merchant.ID.String())
	_ = pgUserID.Scan(merchant.UserID.String())

	created, err := r.queries.CreateMerchantProfile(ctx, db.CreateMerchantProfileParams{
		ID:              pgID,
		UserID:          pgUserID,
		ShopName:        &merchant.ShopName,
		ShopSlug:        &merchant.ShopSlug,
		ShopDescription: merchant.ShopDescription,
		ShopLogoUrl:     merchant.ShopLogoURL,
		ShopBannerUrl:   merchant.ShopBannerURL,
		ContactPhone:    &merchant.ContactPhone,
		ContactEmail:    merchant.ContactEmail,
		Address:         merchant.Address,
		City:            &merchant.City,
		BlnkBalanceID:   merchant.BlnkBalanceID,
		BlnkLedgerID:    merchant.BlnkLedgerID,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to create merchant profile", err)
	}
	return dbMerchantRowToDomain(&created), nil
}

func (r *merchantRepository) Update(ctx context.Context, merchant *domain.MerchantProfile) (*domain.MerchantProfile, error) {
	var pgID pgtype.UUID
	_ = pgID.Scan(merchant.ID.String())

	updated, err := r.queries.UpdateMerchantProfile(ctx, db.UpdateMerchantProfileParams{
		ID:              pgID,
		ShopName:        &merchant.ShopName,
		ShopDescription: merchant.ShopDescription,
		ShopLogoUrl:     merchant.ShopLogoURL,
		ShopBannerUrl:   merchant.ShopBannerURL,
		ContactPhone:    &merchant.ContactPhone,
		ContactEmail:    merchant.ContactEmail,
		Address:         merchant.Address,
		City:            &merchant.City,
		BlnkBalanceID:   merchant.BlnkBalanceID,
		BlnkLedgerID:    merchant.BlnkLedgerID,
		IsVerified:      &merchant.IsVerified,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to update merchant profile", err)
	}
	return dbMerchantRowToDomain(&updated), nil
}

func (r *merchantRepository) UpdateBlnkWallet(ctx context.Context, id uuid.UUID, balanceID, ledgerID string) (*domain.MerchantProfile, error) {
	var pgID pgtype.UUID
	_ = pgID.Scan(id.String())

	updated, err := r.queries.UpdateMerchantBlnkWallet(ctx, db.UpdateMerchantBlnkWalletParams{
		ID:            pgID,
		BlnkBalanceID: &balanceID,
		BlnkLedgerID:  &ledgerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerrors.MerchantNotFound("merchant not found")
		}
		return nil, domainerrors.Internal("failed to update merchant blnk wallet", err)
	}
	return dbMerchantRowToDomain(&updated), nil
}

func (r *merchantRepository) List(ctx context.Context, opts domain.ListOptions) (*domain.PaginatedResult[domain.MerchantProfile], error) {
	limit := int64(opts.Limit())
	offset := int64(opts.Offset())

	merchants, err := r.queries.ListMerchantProfiles(ctx, db.ListMerchantProfilesParams{
		PageLimit:  &limit,
		PageOffset: &offset,
	})
	if err != nil {
		return nil, domainerrors.Internal("failed to list merchants", err)
	}

	items := make([]domain.MerchantProfile, len(merchants))
	for i, m := range merchants {
		items[i] = *dbMerchantRowToDomain(&m)
	}

	return &domain.PaginatedResult[domain.MerchantProfile]{
		Items:   items,
		Total:   int64(len(items)),
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}, nil
}

func dbMerchantRowToDomain[T db.GetMerchantProfileByIDRow | db.GetMerchantProfileByUserIDRow | db.GetMerchantProfileBySlugRow | db.CreateMerchantProfileRow | db.UpdateMerchantProfileRow | db.UpdateMerchantBlnkWalletRow | db.ListMerchantProfilesRow](m *T) *domain.MerchantProfile {
	var id, userID uuid.UUID
	var shopName, shopSlug, contactPhone, city string
	var shopDescription, shopLogoURL, shopBannerURL, contactEmail, address, blnkBalanceID, blnkLedgerID *string
	var isVerified bool
	var totalSalesCents int64
	var createdAt, updatedAt pgtype.Timestamptz

	switch v := any(m).(type) {
	case *db.GetMerchantProfileByIDRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.GetMerchantProfileByUserIDRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.GetMerchantProfileBySlugRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.CreateMerchantProfileRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.UpdateMerchantProfileRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.UpdateMerchantBlnkWalletRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	case *db.ListMerchantProfilesRow:
		id = v.ID
		userID = v.UserID
		shopName = v.ShopName
		shopSlug = v.ShopSlug
		shopDescription = v.ShopDescription
		shopLogoURL = v.ShopLogoUrl
		shopBannerURL = v.ShopBannerUrl
		contactPhone = v.ContactPhone
		contactEmail = v.ContactEmail
		address = v.Address
		city = v.City
		blnkBalanceID = v.BlnkBalanceID
		blnkLedgerID = v.BlnkLedgerID
		isVerified = v.IsVerified
		totalSalesCents = v.TotalSalesCents
		createdAt = v.CreatedAt
		updatedAt = v.UpdatedAt
	}

	return &domain.MerchantProfile{
		ID:              id,
		UserID:          userID,
		ShopName:        shopName,
		ShopSlug:        shopSlug,
		ShopDescription: shopDescription,
		ShopLogoURL:     shopLogoURL,
		ShopBannerURL:   shopBannerURL,
		ContactPhone:    contactPhone,
		ContactEmail:    contactEmail,
		Address:         address,
		City:            city,
		BlnkBalanceID:   blnkBalanceID,
		BlnkLedgerID:    blnkLedgerID,
		IsVerified:      isVerified,
		TotalSalesCents: totalSalesCents,
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}
}
