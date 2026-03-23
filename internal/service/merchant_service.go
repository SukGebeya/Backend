package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/SukGebeya/Backend/internal/domain"
	domainerrors "github.com/SukGebeya/Backend/internal/errors"
	"github.com/SukGebeya/Backend/internal/repository"
)

var _ MerchantOnboarder = (*MerchantService)(nil)

type MerchantService struct {
	merchantRepo repository.MerchantRepository
}

func NewMerchantService(merchantRepo repository.MerchantRepository) *MerchantService {
	return &MerchantService{
		merchantRepo: merchantRepo,
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
}

func (s *MerchantService) Onboard(ctx context.Context, input OnboardInput) (*domain.MerchantProfile, error) {
	slug := input.ShopSlug
	if slug == "" {
		slug = generateSlug(input.ShopName)
	}

	existing, _ := s.merchantRepo.GetBySlug(ctx, slug)
	if existing != nil {
		return nil, domainerrors.ShopSlugExists("shop slug already exists")
	}

	existingProfile, _ := s.merchantRepo.GetByUserID(ctx, input.UserID)
	if existingProfile != nil {
		return nil, domainerrors.Conflict("user already has a merchant profile")
	}

	merchant := &domain.MerchantProfile{
		ID:              uuid.New(),
		UserID:          input.UserID,
		ShopName:        input.ShopName,
		ShopSlug:        slug,
		ShopDescription: input.ShopDescription,
		ContactPhone:    input.ContactPhone,
		ContactEmail:    input.ContactEmail,
		Address:         input.Address,
		City:            input.City,
		IsVerified:      false,
		TotalSalesCents: 0,
	}

	created, err := s.merchantRepo.Create(ctx, merchant)
	if err != nil {
		return nil, domainerrors.Internal("failed to create merchant profile", err)
	}

	return created, nil
}

func (s *MerchantService) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MerchantProfile, error) {
	return s.merchantRepo.GetByUserID(ctx, userID)
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
		merchant.ContactPhone = *input.ContactPhone
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
