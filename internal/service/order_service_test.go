package service

import (
	"context"
	"testing"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOrderService_CancelOrder_Success(t *testing.T) {
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
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	category := testutil.NewCategoryBuilder().Build()
	createdCategory, err := categoryRepo.Create(ctx, category)
	require.NoError(t, err)

	product := testutil.NewProductBuilder().
		WithMerchantID(createdMerchant.ID).
		WithCategoryID(createdCategory.ID).
		Build()
	createdProduct, err := productRepo.Create(ctx, product)
	require.NoError(t, err)

	variant := testutil.NewVariantBuilder().
		WithProductID(createdProduct.ID).
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

	err = inventoryRepo.Reserve(ctx, createdVariant.ID, 2)
	require.NoError(t, err)

	invBefore, err := inventoryRepo.GetByVariantID(ctx, createdVariant.ID)
	require.NoError(t, err)
	require.Equal(t, 100, invBefore.Quantity)
	require.Equal(t, 2, invBefore.Reserved)

	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		WithTotal(2000).
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

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	cancelledOrder, err := orderService.CancelOrder(ctx, createdOrder.ID, "buyer")
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusCancelled, cancelledOrder.Status)

	invAfter, err := inventoryRepo.GetByVariantID(ctx, createdVariant.ID)
	require.NoError(t, err)
	require.Equal(t, 100, invAfter.Quantity)
	require.Equal(t, 0, invAfter.Reserved)
}

func TestOrderService_CancelOrder_NotFound(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	orderRepo := repository.NewOrderRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	userRepo := repository.NewUserRepository(pool)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	_, err := orderService.CancelOrder(ctx, uuid.New(), "buyer")
	require.Error(t, err)
}

func TestOrderService_CancelOrder_InvalidStatus(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusDelivered).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	_, err = orderService.CancelOrder(ctx, createdOrder.ID, "buyer")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot cancel order")
}

func TestOrderService_GetByID_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	fetchedOrder, err := orderService.GetByID(ctx, createdOrder.ID)
	require.NoError(t, err)
	require.Equal(t, createdOrder.ID, fetchedOrder.ID)
	require.Equal(t, domain.OrderStatusPendingPayment, fetchedOrder.Status)
}

func TestOrderService_UpdateStatus_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	updatedOrder, err := orderService.UpdateStatus(ctx, createdOrder.ID, domain.OrderStatusPaid)
	require.NoError(t, err)
	require.Equal(t, domain.OrderStatusPaid, updatedOrder.Status)
}

func TestOrderService_UpdateStatus_InvalidTransition(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	order := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		Build()
	createdOrder, err := orderRepo.Create(ctx, order)
	require.NoError(t, err)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	_, err = orderService.UpdateStatus(ctx, createdOrder.ID, domain.OrderStatusDelivered)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot transition")
}

func TestOrderService_ListByBuyerID(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	orderRepo := repository.NewOrderRepository(pool)
	shipmentRepo := repository.NewShipmentRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	order1 := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPendingPayment).
		Build()
	_, err = orderRepo.Create(ctx, order1)
	require.NoError(t, err)

	order2 := testutil.NewOrderBuilder().
		WithBuyerID(createdUser.ID).
		WithMerchantID(createdMerchant.ID).
		WithStatus(domain.OrderStatusPaid).
		Build()
	_, err = orderRepo.Create(ctx, order2)
	require.NoError(t, err)

	orderService := NewOrderService(
		orderRepo,
		merchantRepo,
		shipmentRepo,
		walletRepo,
		nil,
		inventoryRepo,
		userRepo,
		repository.NewTransactionManager(pool),
	)

	result, err := orderService.ListByBuyerID(ctx, createdUser.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Equal(t, int64(2), result.Total)
}
