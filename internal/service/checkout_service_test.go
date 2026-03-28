package service

import (
	"context"
	"testing"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/mocks"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCheckoutService_Checkout_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	txManager := repository.NewTransactionManager(pool)

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	paymentRepo := repository.NewPaymentRepository(pool)
	addressRepo := repository.NewAddressRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	chapaClient := mocks.NewMockChapaClient()

	checkoutService := NewCheckoutService(
		cartRepo,
		cartItemRepo,
		orderRepo,
		paymentRepo,
		addressRepo,
		inventoryRepo,
		merchantRepo,
		productRepo,
		variantRepo,
		blnkClient,
		chapaClient,
		txManager,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	category := testutil.NewCategoryBuilder().Build()
	createdCategory, err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	product := testutil.NewProductBuilder().
		WithMerchantID(createdMerchant.ID).
		WithCategoryID(createdCategory.ID).
		Build()
	createdProduct, err := productRepo.Create(ctx, product)
	require.NoError(t, err)

	variant := testutil.NewVariantBuilder().
		WithProductID(createdProduct.ID).
		WithPrice(1000).
		Build()
	createdVariant, err := variantRepo.Create(ctx, variant)
	require.NoError(t, err)

	inventory := testutil.NewInventoryBuilder().
		WithVariantID(createdVariant.ID).
		WithQuantity(100).
		WithReserved(0).
		Build()
	_, err = inventoryRepo.Create(ctx, inventory)
	require.NoError(t, err)

	cart := testutil.NewCartBuilder().
		WithUserID(createdUser.ID).
		Build()
	createdCart, err := cartRepo.Create(ctx, cart)
	require.NoError(t, err)

	cartItem := testutil.NewCartItemBuilder().
		WithCartID(createdCart.ID).
		WithVariantID(createdVariant.ID).
		WithQuantity(2).
		WithPrice(1000).
		Build()
	_, err = cartItemRepo.Create(ctx, cartItem)
	require.NoError(t, err)

	result, err := checkoutService.Checkout(ctx, CheckoutInput{
		UserID: createdUser.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEqual(t, uuid.Nil, result.OrderID)
	require.Equal(t, int64(2000), result.TotalCents)
	require.Equal(t, "ETB", result.Currency)
	require.NotEmpty(t, result.PaymentURL)
	require.NotEmpty(t, result.ChapaTxRef)

	invAfter, err := inventoryRepo.GetByVariantID(ctx, createdVariant.ID)
	require.NoError(t, err)
	require.Equal(t, 100, invAfter.Quantity)
	require.Equal(t, 2, invAfter.Reserved)
}

func TestCheckoutService_ProcessSuccessfulPayment(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	paymentRepo := repository.NewPaymentRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	chapaClient := mocks.NewMockChapaClient()

	checkoutService := NewCheckoutService(
		cartRepo,
		cartItemRepo,
		orderRepo,
		paymentRepo,
		nil,
		inventoryRepo,
		merchantRepo,
		productRepo,
		variantRepo,
		blnkClient,
		chapaClient,
		nil,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	category := testutil.NewCategoryBuilder().Build()
	createdCategory, err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	product := testutil.NewProductBuilder().
		WithMerchantID(createdMerchant.ID).
		WithCategoryID(createdCategory.ID).
		Build()
	createdProduct, err := productRepo.Create(ctx, product)
	require.NoError(t, err)

	variant := testutil.NewVariantBuilder().
		WithProductID(createdProduct.ID).
		WithPrice(1000).
		Build()
	createdVariant, err := variantRepo.Create(ctx, variant)
	require.NoError(t, err)

	inventory := testutil.NewInventoryBuilder().
		WithVariantID(createdVariant.ID).
		WithQuantity(100).
		WithReserved(2).
		Build()
	_, err = inventoryRepo.Create(ctx, inventory)
	require.NoError(t, err)

	chapaTxRef := "GEBEYA-test-" + uuid.New().String()
	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		WithChapaTxRef(chapaTxRef).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderItem := testutil.NewOrderItemBuilder().
		WithOrderID(createdOrder.ID).
		WithVariantID(createdVariant.ID).
		WithQuantity(2).
		WithUnitPrice(1000).
		Build()
	_, err = orderRepo.CreateItem(ctx, orderItem)
	require.NoError(t, err)

	payment := testutil.NewPaymentBuilder().
		WithOrderID(createdOrder.ID).
		WithStatus(domain.PaymentStatusPending).
		Build()
	_, err = paymentRepo.Create(ctx, payment)
	require.NoError(t, err)

	err = checkoutService.ProcessSuccessfulPayment(ctx, chapaTxRef)
	require.NoError(t, err)

	updatedOrder, err := orderRepo.GetByID(ctx, createdOrder.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPaid, updatedOrder.Status)

	updatedPayment, err := paymentRepo.GetByOrderID(ctx, createdOrder.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusSucceeded, updatedPayment.Status)

	invAfter, err := inventoryRepo.GetByVariantID(ctx, createdVariant.ID)
	require.NoError(t, err)
	require.Equal(t, 98, invAfter.Quantity)
	require.Equal(t, 0, invAfter.Reserved)
}

func TestCheckoutService_ProcessFailedPayment(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	categoryRepo := repository.NewCategoryRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	variantRepo := repository.NewProductVariantRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	cartRepo := repository.NewCartRepository(pool)
	cartItemRepo := repository.NewCartItemRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	paymentRepo := repository.NewPaymentRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	chapaClient := mocks.NewMockChapaClient()

	checkoutService := NewCheckoutService(
		cartRepo,
		cartItemRepo,
		orderRepo,
		paymentRepo,
		nil,
		inventoryRepo,
		merchantRepo,
		productRepo,
		variantRepo,
		blnkClient,
		chapaClient,
		nil,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	category := testutil.NewCategoryBuilder().Build()
	createdCategory, err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	product := testutil.NewProductBuilder().
		WithMerchantID(createdMerchant.ID).
		WithCategoryID(createdCategory.ID).
		Build()
	createdProduct, err := productRepo.Create(ctx, product)
	require.NoError(t, err)

	variant := testutil.NewVariantBuilder().
		WithProductID(createdProduct.ID).
		WithPrice(1000).
		Build()
	createdVariant, err := variantRepo.Create(ctx, variant)
	require.NoError(t, err)

	inventory := testutil.NewInventoryBuilder().
		WithVariantID(createdVariant.ID).
		WithQuantity(100).
		WithReserved(2).
		Build()
	_, err = inventoryRepo.Create(ctx, inventory)
	require.NoError(t, err)

	chapaTxRef := "GEBEYA-test-" + uuid.New().String()
	blnkTxID := "inflight-" + uuid.New().String()
	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		WithChapaTxRef(chapaTxRef).
		WithBlnkTransactionID(blnkTxID).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderItem := testutil.NewOrderItemBuilder().
		WithOrderID(createdOrder.ID).
		WithVariantID(createdVariant.ID).
		WithQuantity(2).
		WithUnitPrice(1000).
		Build()
	_, err = orderRepo.CreateItem(ctx, orderItem)
	require.NoError(t, err)

	payment := testutil.NewPaymentBuilder().
		WithOrderID(createdOrder.ID).
		WithStatus(domain.PaymentStatusPending).
		Build()
	_, err = paymentRepo.Create(ctx, payment)
	require.NoError(t, err)

	err = checkoutService.ProcessFailedPayment(ctx, chapaTxRef)
	require.NoError(t, err)

	updatedOrder, err := orderRepo.GetByID(ctx, createdOrder.ID)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, updatedOrder.Status)

	updatedPayment, err := paymentRepo.GetByOrderID(ctx, createdOrder.ID)
	require.NoError(t, err)
	require.Equal(t, domain.PaymentStatusFailed, updatedPayment.Status)

	invAfter, err := inventoryRepo.GetByVariantID(ctx, createdVariant.ID)
	require.NoError(t, err)
	require.Equal(t, 100, invAfter.Quantity)
	require.Equal(t, 0, invAfter.Reserved)
}
