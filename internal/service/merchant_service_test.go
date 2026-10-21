package service_test

import (
	"context"
	"testing"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/mocks"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
)

func TestMerchantService_Onboard_CreatesBlnkWallet(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)

	var walletShopNames []string
	mockBlnkClient := mocks.NewMockBlnkClient()
	mockBlnkClient.CreateMerchantWalletFunc = func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
		walletShopNames = append(walletShopNames, shopName)
		return "blnk_balance_test123", nil
	}

	merchantService := service.NewMerchantService(merchantRepo, userRepo, mockBlnkClient, false)

	userBuilder := testutil.NewUserBuilder().
		WithEmail("merchant@example.com").
		WithPhone("+251911111111")
	user := userBuilder.Build()

	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	onboardInput := service.OnboardInput{
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
	}

	merchant, err := merchantService.Onboard(ctx, onboardInput)
	if err != nil {
		t.Fatalf("Onboard() failed: %v", err)
	}

	if got := mockBlnkClient.CallCount("CreateMerchantWallet"); got != 1 {
		t.Fatalf("expected 1 CreateMerchantWallet call, got %d", got)
	}

	if walletShopNames[0] != onboardInput.ShopName {
		t.Errorf("expected shop name %q, got %q", onboardInput.ShopName, walletShopNames[0])
	}

	if merchant.BlnkBalanceID == nil || *merchant.BlnkBalanceID == "" {
		t.Error("expected BlnkBalanceID to be non-empty after Onboard")
	}

	if *merchant.BlnkBalanceID != "blnk_balance_test123" {
		t.Errorf("expected BlnkBalanceID %q, got %q", "blnk_balance_test123", *merchant.BlnkBalanceID)
	}
}

func TestMerchantService_Onboard_DuplicateSlug(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)

	mockBlnkClient := mocks.NewMockBlnkClient()

	merchantService := service.NewMerchantService(merchantRepo, userRepo, mockBlnkClient, false)

	user1Builder := testutil.NewUserBuilder().WithEmail("user1@example.com")
	user1 := user1Builder.Build()
	createdUser1, err := userRepo.Create(ctx, user1)
	if err != nil {
		t.Fatalf("failed to create test user1: %v", err)
	}

	onboardInput1 := service.OnboardInput{
		UserID:       createdUser1.ID,
		ShopName:     "Shop One",
		ShopSlug:     "duplicate-slug",
		ContactPhone: "+251911111111",
		City:         "Addis Ababa",
	}

	_, err = merchantService.Onboard(ctx, onboardInput1)
	if err != nil {
		t.Fatalf("first Onboard() failed: %v", err)
	}

	user2Builder := testutil.NewUserBuilder().WithEmail("user2@example.com").WithPhone("+251922222221")
	user2 := user2Builder.Build()
	createdUser2, err := userRepo.Create(ctx, user2)
	if err != nil {
		t.Fatalf("failed to create test user2: %v", err)
	}

	onboardInput2 := service.OnboardInput{
		UserID:       createdUser2.ID,
		ShopName:     "Shop Two",
		ShopSlug:     "duplicate-slug",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
	}

	_, err = merchantService.Onboard(ctx, onboardInput2)
	if err == nil {
		t.Error("expected error for duplicate slug, got nil")
	}
}

func TestMerchantService_Onboard_AlreadyHasProfile(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)

	mockBlnkClient := mocks.NewMockBlnkClient()

	merchantService := service.NewMerchantService(merchantRepo, userRepo, mockBlnkClient, false)

	userBuilder := testutil.NewUserBuilder().WithEmail("user@example.com")
	user := userBuilder.Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	onboardInput := service.OnboardInput{
		UserID:       createdUser.ID,
		ShopName:     "First Shop",
		ShopSlug:     "first-shop",
		ContactPhone: "+251911111111",
		City:         "Addis Ababa",
	}

	_, err = merchantService.Onboard(ctx, onboardInput)
	if err != nil {
		t.Fatalf("first Onboard() failed: %v", err)
	}

	onboardInput2 := service.OnboardInput{
		UserID:       createdUser.ID,
		ShopName:     "Second Shop",
		ShopSlug:     "second-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
	}

	_, err = merchantService.Onboard(ctx, onboardInput2)
	if err == nil {
		t.Error("expected error for user with existing profile, got nil")
	}
}

func TestMerchantProfile_HasWallet(t *testing.T) {
	emptyStr := ""
	nonEmptyStr := "blnk_balance_123"

	tests := []struct {
		name     string
		profile  domain.MerchantProfile
		expected bool
	}{
		{
			name: "nil BlnkBalanceID returns false",
			profile: domain.MerchantProfile{
				BlnkBalanceID: nil,
			},
			expected: false,
		},
		{
			name: "empty BlnkBalanceID returns false",
			profile: domain.MerchantProfile{
				BlnkBalanceID: &emptyStr,
			},
			expected: false,
		},
		{
			name: "non-empty BlnkBalanceID returns true",
			profile: domain.MerchantProfile{
				BlnkBalanceID: &nonEmptyStr,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.profile.HasWallet()
			if result != tt.expected {
				t.Errorf("HasWallet() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
