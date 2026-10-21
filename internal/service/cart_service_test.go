package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/service"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
)

func TestCartService_GetOrCreateCart_UserCart(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	userBuilder := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111")
	user := userBuilder.Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("GetOrCreateCart() failed: %v", err)
	}

	if cart == nil {
		t.Fatal("expected cart, got nil")
	}

	if cart.UserID == nil || *cart.UserID != createdUser.ID {
		t.Errorf("expected cart user ID %v, got %v", createdUser.ID, cart.UserID)
	}

	if cart.Status != domain.CartStatusActive {
		t.Errorf("expected cart status %s, got %s", domain.CartStatusActive, cart.Status)
	}
}

func TestCartService_GetOrCreateCart_SessionCart(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	sessionID := "test-session-123"
	cart, err := cartService.GetOrCreateCart(ctx, nil, &sessionID)
	if err != nil {
		t.Fatalf("GetOrCreateCart() failed: %v", err)
	}

	if cart == nil {
		t.Fatal("expected cart, got nil")
	}

	if cart.SessionID == nil || *cart.SessionID != sessionID {
		t.Errorf("expected cart session ID %s, got %v", sessionID, cart.SessionID)
	}

	if cart.Status != domain.CartStatusActive {
		t.Errorf("expected cart status %s, got %s", domain.CartStatusActive, cart.Status)
	}
}

func TestCartService_GetOrCreateCart_ReturnsExistingCart(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	userBuilder := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111")
	user := userBuilder.Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	cart1, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("first GetOrCreateCart() failed: %v", err)
	}

	cart2, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("second GetOrCreateCart() failed: %v", err)
	}

	if cart1.ID != cart2.ID {
		t.Errorf("expected same cart ID, got %v and %v", cart1.ID, cart2.ID)
	}
}

func TestCartService_AddItem(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail("merchant@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Default",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant, err := variantRepo.Create(ctx, variant)
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}

	item, err := cartService.AddItem(ctx, cart.ID, createdVariant.ID, 2, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("AddItem() failed: %v", err)
	}

	if item == nil {
		t.Fatal("expected item, got nil")
	}

	if item.VariantID != createdVariant.ID {
		t.Errorf("expected variant ID %v, got %v", createdVariant.ID, item.VariantID)
	}

	if item.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", item.Quantity)
	}

	if item.PriceSnapshotCents != createdVariant.PriceCents {
		t.Errorf("expected price %d, got %d", createdVariant.PriceCents, item.PriceSnapshotCents)
	}
}

func TestCartService_AddItem_MergeQuantity(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail(" merchant@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Default",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant, err := variantRepo.Create(ctx, variant)
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}

	_, err = cartService.AddItem(ctx, cart.ID, createdVariant.ID, 3, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("first AddItem() failed: %v", err)
	}

	item2, err := cartService.AddItem(ctx, cart.ID, createdVariant.ID, 2, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("second AddItem() failed: %v", err)
	}

	if item2.Quantity != 5 {
		t.Errorf("expected merged quantity 5, got %d", item2.Quantity)
	}
}

func TestCartService_MergeCart(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Default",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant, err := variantRepo.Create(ctx, variant)
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	sessionID := "test-session-merge"
	sessionCart, err := cartService.GetOrCreateCart(ctx, nil, &sessionID)
	if err != nil {
		t.Fatalf("failed to create session cart: %v", err)
	}

	_, err = cartService.AddItem(ctx, sessionCart.ID, createdVariant.ID, 5, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("failed to add item to session cart: %v", err)
	}

	mergedCart, err := cartService.MergeCart(ctx, createdUser.ID, sessionID)
	if err != nil {
		t.Fatalf("MergeCart() failed: %v", err)
	}

	if mergedCart == nil {
		t.Fatal("expected merged cart, got nil")
	}

	if len(mergedCart.Items) != 1 {
		t.Errorf("expected 1 item in merged cart, got %d", len(mergedCart.Items))
	}

	if len(mergedCart.Items) > 0 && mergedCart.Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5 in merged item, got %d", mergedCart.Items[0].Quantity)
	}

	if mergedCart.UserID == nil || *mergedCart.UserID != createdUser.ID {
		t.Errorf("expected merged cart to belong to user %v", createdUser.ID)
	}
}

func TestCartService_UpdateItemQuantityByVariant(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Default",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant, err := variantRepo.Create(ctx, variant)
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}

	_, err = cartService.AddItem(ctx, cart.ID, createdVariant.ID, 3, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("AddItem() failed: %v", err)
	}

	updatedItem, err := cartService.UpdateItemQuantityByVariant(ctx, cart.ID, createdVariant.ID, 10)
	if err != nil {
		t.Fatalf("UpdateItemQuantityByVariant() failed: %v", err)
	}

	if updatedItem == nil {
		t.Fatal("expected updated item, got nil")
	}

	if updatedItem.Quantity != 10 {
		t.Errorf("expected quantity 10, got %d", updatedItem.Quantity)
	}

	removedItem, err := cartService.UpdateItemQuantityByVariant(ctx, cart.ID, createdVariant.ID, 0)
	if err != nil {
		t.Fatalf("UpdateItemQuantityByVariant() with 0 failed: %v", err)
	}

	if removedItem != nil {
		t.Errorf("expected nil when removing item with quantity 0, got item with quantity %d", removedItem.Quantity)
	}
}

func TestCartService_RemoveItemByVariant(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Default",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant, err := variantRepo.Create(ctx, variant)
	if err != nil {
		t.Fatalf("failed to create variant: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}

	_, err = cartService.AddItem(ctx, cart.ID, createdVariant.ID, 5, createdVariant.PriceCents)
	if err != nil {
		t.Fatalf("AddItem() failed: %v", err)
	}

	err = cartService.RemoveItemByVariant(ctx, cart.ID, createdVariant.ID)
	if err != nil {
		t.Fatalf("RemoveItemByVariant() failed: %v", err)
	}

	cartWithItems, err := cartService.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		t.Fatalf("GetCartWithItems() failed: %v", err)
	}

	if len(cartWithItems.Items) != 0 {
		t.Errorf("expected 0 items after removal, got %d", len(cartWithItems.Items))
	}
}

func TestCartService_ClearCart(t *testing.T) {
	ctx := context.Background()

	pool := testutil.NewTestDatabase(t)

	redisClient := testutil.NewTestRedis(t)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	imageRepo := repository.NewProductImageRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	sessionRepo := repository.NewSessionRepository(redisClient)

	cartService := service.NewCartService(cartRepo, cartItemRepo, sessionRepo, variantRepo, inventoryRepo, nil)

	user := testutil.NewUserBuilder().WithEmail("user@example.com").WithPhone("+251911111111").Build()
	createdUser, err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	merchant := &domain.MerchantProfile{
		ID:           uuid.New(),
		UserID:       createdUser.ID,
		ShopName:     "Test Shop",
		ShopSlug:     "test-shop",
		ContactPhone: "+251922222222",
		City:         "Addis Ababa",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	if err != nil {
		t.Fatalf("failed to create merchant: %v", err)
	}

	category := &domain.Category{
		ID:        uuid.New(),
		Name:      "Test Category",
		Slug:      "test-category",
		CreatedAt: time.Now(),
	}
	createdCategory, err := categoryRepo.Create(ctx, category)
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	product := &domain.Product{
		ID:         uuid.New(),
		MerchantID: createdMerchant.ID,
		CategoryID: &createdCategory.ID,
		Name:       "Test Product",
		Slug:       "test-product",
		Status:     domain.ProductStatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdProduct, err := productRepo.Create(ctx, product)
	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	_ = imageRepo

	variant1 := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-001",
		Title:      "Variant 1",
		PriceCents: 1000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant1, err := variantRepo.Create(ctx, variant1)
	if err != nil {
		t.Fatalf("failed to create variant1: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant1.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	variant2 := &domain.ProductVariant{
		ID:         uuid.New(),
		ProductID:  createdProduct.ID,
		SKU:        "SKU-002",
		Title:      "Variant 2",
		PriceCents: 2000,
		Currency:   "ETB",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	createdVariant2, err := variantRepo.Create(ctx, variant2)
	if err != nil {
		t.Fatalf("failed to create variant2: %v", err)
	}
	if _, err := inventoryRepo.Create(ctx, &domain.Inventory{ID: uuid.New(), VariantID: createdVariant2.ID, Quantity: 100}); err != nil {
		t.Fatalf("failed to stock variant: %v", err)
	}

	cart, err := cartService.GetOrCreateCart(ctx, &createdUser.ID, nil)
	if err != nil {
		t.Fatalf("failed to create cart: %v", err)
	}

	_, err = cartService.AddItem(ctx, cart.ID, createdVariant1.ID, 3, createdVariant1.PriceCents)
	if err != nil {
		t.Fatalf("AddItem() for variant1 failed: %v", err)
	}

	_, err = cartService.AddItem(ctx, cart.ID, createdVariant2.ID, 2, createdVariant2.PriceCents)
	if err != nil {
		t.Fatalf("AddItem() for variant2 failed: %v", err)
	}

	cartWithItems, err := cartService.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		t.Fatalf("GetCartWithItems() failed: %v", err)
	}

	if len(cartWithItems.Items) != 2 {
		t.Errorf("expected 2 items before clear, got %d", len(cartWithItems.Items))
	}

	err = cartService.ClearCart(ctx, cart.ID)
	if err != nil {
		t.Fatalf("ClearCart() failed: %v", err)
	}

	cartWithItems, err = cartService.GetCartWithItems(ctx, cart.ID)
	if err != nil {
		t.Fatalf("GetCartWithItems() after clear failed: %v", err)
	}

	if len(cartWithItems.Items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(cartWithItems.Items))
	}
}
