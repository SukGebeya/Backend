package service

import (
	"context"
	"log/slog"
	"strings"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/pkg/phone"
	"github.com/google/uuid"
)

var _ MerchantOnboarder = (*MerchantService)(nil)

type MerchantService struct {
	merchantRepo             repository.MerchantRepository
	userRepo                 repository.UserRepository
	blnkClient               blnk.BlnkClient
	requirePhoneVerification bool
}

func NewMerchantService(merchantRepo repository.MerchantRepository, userRepo repository.UserRepository, blnkClient blnk.BlnkClient, requirePhoneVerification bool) *MerchantService {
	return &MerchantService{
		merchantRepo:             merchantRepo,
		userRepo:                 userRepo,
		blnkClient:               blnkClient,
		requirePhoneVerification: requirePhoneVerification,
	}
}

type OnboardInput struct {
	UserID          uuid.UUID
	ShopName        string
	ShopSlug        string
	ShopDescription *string
	ContactPhone    string
	ContactEmail    *string
	Address         *string
	City            string

	DeliveryFeeCents           int64
	FreeDeliveryThresholdCents *int64
	TIN                        *string
	TradeLicenseNumber         *string
}

type OnboardResult struct {
	Merchant      *domain.MerchantProfile
	BlnkBalanceID string
	Message       string
}

type UpdateMerchantInput struct {
	ShopName        *string
	ShopDescription *string
	ShopLogoURL     *string
	ShopBannerURL   *string
	ContactPhone    *string
	ContactEmail    *string
	Address         *string
	City            *string

	DeliveryFeeCents           *int64
	FreeDeliveryThresholdCents *int64
	ClearFreeDeliveryThreshold bool
	TIN                        *string
	TradeLicenseNumber         *string
}

func (s *MerchantService) Onboard(ctx context.Context, input OnboardInput) (*domain.MerchantProfile, error) {
	existingProfile, _ := s.merchantRepo.GetByUserID(ctx, input.UserID)
	if existingProfile != nil {
		return nil, domainerrors.Conflict("user already has a merchant profile")
	}

	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if s.requirePhoneVerification && user.PhoneVerifiedAt == nil {
		return nil, domainerrors.Forbidden("verify your phone number before opening a shop")
	}

	contactPhone, err := phone.Normalize(input.ContactPhone)
	if err != nil {
		return nil, domainerrors.Validation("contact_phone must be an Ethiopian mobile number")
	}
	input.ContactPhone = contactPhone

	if input.DeliveryFeeCents < 0 {
		return nil, domainerrors.Validation("delivery_fee_cents cannot be negative")
	}

	slug := input.ShopSlug
	if slug == "" {
		slug, err = s.uniqueSlug(ctx, input.ShopName)
		if err != nil {
			return nil, err
		}
	} else if existing, _ := s.merchantRepo.GetBySlug(ctx, slug); existing != nil {
		return nil, domainerrors.ShopSlugExists("shop slug already exists")
	}

	merchantID := uuid.New()
	balanceID, err := s.blnkClient.CreateMerchantWallet(ctx, merchantID, input.ShopName)
	if err != nil {
		slog.Error("Onboard: failed to create Blnk wallet",
			"error", err,
			"user_id", input.UserID.String(),
			"shop_name", input.ShopName,
		)
		return nil, domainerrors.Internal("failed to create merchant wallet", err)
	}

	slog.Debug("Onboard: created Blnk wallet",
		"balance_id", balanceID,
		"user_id", input.UserID.String(),
	)

	ledgerID := s.blnkClient.LedgerID()
	merchant := &domain.MerchantProfile{
		ID:              merchantID,
		UserID:          input.UserID,
		ShopName:        input.ShopName,
		ShopSlug:        slug,
		ShopDescription: input.ShopDescription,
		ContactPhone:    input.ContactPhone,
		ContactEmail:    input.ContactEmail,
		Address:         input.Address,
		City:            input.City,
		BlnkBalanceID:   &balanceID,
		BlnkLedgerID:    &ledgerID,
		IsVerified:      false,
		TotalSalesCents: 0,

		DeliveryFeeCents:           input.DeliveryFeeCents,
		FreeDeliveryThresholdCents: input.FreeDeliveryThresholdCents,
		TIN:                        input.TIN,
		TradeLicenseNumber:         input.TradeLicenseNumber,
	}

	created, err := s.merchantRepo.Create(ctx, merchant)
	if err != nil {
		slog.Error("Onboard: failed to create merchant profile after Blnk wallet creation",
			"error", err,
			"user_id", input.UserID.String(),
			"balance_id", balanceID,
		)
		return nil, domainerrors.Internal("failed to create merchant profile", err)
	}

	if user.Role == domain.RoleBuyer {
		user.Role = domain.RoleMerchant
		if _, err := s.userRepo.Update(ctx, user); err != nil {
			slog.Error("Onboard: failed to promote user to merchant role", "user_id", user.ID, "error", err)
		}
	}

	return created, nil
}

// ProvisionWallet creates a Blnk wallet for a merchant that has none, for example
// after seeding or when Blnk was unreachable during onboarding.
func (s *MerchantService) ProvisionWallet(ctx context.Context, merchantID uuid.UUID) (*domain.MerchantProfile, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}
	if merchant.BlnkBalanceID != nil && *merchant.BlnkBalanceID != "" {
		return merchant, nil
	}

	balanceID, err := s.blnkClient.CreateMerchantWallet(ctx, merchant.ID, merchant.ShopName)
	if err != nil {
		return nil, domainerrors.Internal("failed to create merchant wallet", err)
	}
	return s.merchantRepo.UpdateBlnkWallet(ctx, merchant.ID, balanceID, s.blnkClient.LedgerID())
}

func (s *MerchantService) uniqueSlug(ctx context.Context, name string) (string, error) {
	base := generateSlug(name)
	slug := base
	for i := 0; i < 5; i++ {
		existing, err := s.merchantRepo.GetBySlug(ctx, slug)
		if err != nil && !domainerrors.IsNotFound(err) {
			return "", err
		}
		if existing == nil {
			return slug, nil
		}
		slug = base + "-" + uuid.NewString()[:4]
	}
	return base + "-" + uuid.NewString()[:8], nil
}

func (s *MerchantService) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error) {
	return s.merchantRepo.GetByUserID(ctx, userID)
}

func (s *MerchantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.MerchantProfile, error) {
	return s.merchantRepo.GetByID(ctx, id)
}

func (s *MerchantService) GetBySlug(ctx context.Context, slug string) (*domain.MerchantProfile, error) {
	return s.merchantRepo.GetBySlug(ctx, slug)
}

func (s *MerchantService) Update(ctx context.Context, merchantID uuid.UUID, input UpdateMerchantInput) (*domain.MerchantProfile, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	if input.ShopName != nil {
		merchant.ShopName = *input.ShopName
	}
	if input.ShopDescription != nil {
		merchant.ShopDescription = input.ShopDescription
	}
	if input.ShopLogoURL != nil {
		merchant.ShopLogoURL = input.ShopLogoURL
	}
	if input.ShopBannerURL != nil {
		merchant.ShopBannerURL = input.ShopBannerURL
	}
	if input.ContactPhone != nil {
		normalized, err := phone.Normalize(*input.ContactPhone)
		if err != nil {
			return nil, domainerrors.Validation("contact_phone must be an Ethiopian mobile number")
		}
		merchant.ContactPhone = normalized
	}
	if input.ContactEmail != nil {
		merchant.ContactEmail = input.ContactEmail
	}
	if input.Address != nil {
		merchant.Address = input.Address
	}
	if input.City != nil {
		merchant.City = *input.City
	}
	if input.DeliveryFeeCents != nil {
		if *input.DeliveryFeeCents < 0 {
			return nil, domainerrors.Validation("delivery_fee_cents cannot be negative")
		}
		merchant.DeliveryFeeCents = *input.DeliveryFeeCents
	}
	if input.ClearFreeDeliveryThreshold {
		merchant.FreeDeliveryThresholdCents = nil
	} else if input.FreeDeliveryThresholdCents != nil {
		merchant.FreeDeliveryThresholdCents = input.FreeDeliveryThresholdCents
	}
	if input.TIN != nil {
		merchant.TIN = input.TIN
	}
	if input.TradeLicenseNumber != nil {
		merchant.TradeLicenseNumber = input.TradeLicenseNumber
	}

	return s.merchantRepo.Update(ctx, merchant)
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result []rune
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}

	slug = string(result)
	slug = strings.Trim(slug, "-")

	if len(slug) > 50 {
		slug = slug[:50]
	}

	if slug == "" {
		slug = uuid.New().String()[:8]
	}

	return slug
}

func (s *MerchantService) Verify(ctx context.Context, merchantID uuid.UUID, isVerified bool) (*domain.MerchantProfile, error) {
	merchant, err := s.merchantRepo.GetByID(ctx, merchantID)
	if err != nil {
		return nil, err
	}

	merchant.IsVerified = isVerified

	return s.merchantRepo.Update(ctx, merchant)
}
