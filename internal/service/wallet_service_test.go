package service

import (
	"context"
	"errors"
	"testing"

	"github.com/SukGebeya/Backend/internal/domain"
	"github.com/SukGebeya/Backend/internal/mocks"
	"github.com/SukGebeya/Backend/internal/repository"
	"github.com/SukGebeya/Backend/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletService_GetBalance_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	blnkClient.GetBalanceFunc = func(ctx context.Context, balanceID string) (int64, error) {
		return 150000, nil // 1500.00 ETB
	}

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	blnkBalanceID := "bln_test123"
	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		WithBlnkBalanceID(blnkBalanceID).
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	balance, err := walletService.GetBalance(ctx, createdMerchant.ID)
	require.NoError(t, err)
	require.NotNil(t, balance)
	require.Equal(t, int64(150000), balance.BalanceCents)
	require.Equal(t, "ETB", balance.Currency)
}

func TestWalletService_GetBalance_NoWallet(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	balance, err := walletService.GetBalance(ctx, createdMerchant.ID)
	require.NoError(t, err)
	require.NotNil(t, balance)
	require.Equal(t, int64(0), balance.BalanceCents)
	require.Equal(t, "ETB", balance.Currency)
}

func TestWalletService_GetBalance_BlnkError(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	blnkClient.GetBalanceFunc = func(ctx context.Context, balanceID string) (int64, error) {
		return 0, errors.New("blnk service unavailable")
	}

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		WithBlnkBalanceID("bln_test123").
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	_, err = walletService.GetBalance(ctx, createdMerchant.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get wallet balance")
}

func TestWalletService_GetTransactionHistory_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	description := "Test transaction"
	txn := &domain.WalletTransaction{
		ID:                uuid.New(),
		MerchantID:        createdMerchant.ID,
		OrderID:           nil,
		Type:              domain.WalletTransactionTypeSaleCredit,
		AmountCents:       1000,
		Currency:          "ETB",
		BlnkTransactionID: "blnk_txn_123",
		Status:            domain.WalletTransactionStatusApplied,
		Description:       &description,
	}
	_, err = walletRepo.Create(ctx, txn)
	require.NoError(t, err)

	page := 1
	pageSize := 10
	result, err := walletService.GetTransactionHistory(ctx, createdMerchant.ID, page, pageSize)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, int64(1000), result.Items[0].AmountCents)
}

func TestWalletService_GetTransactionHistory_Empty(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	page := 1
	pageSize := 10
	result, err := walletService.GetTransactionHistory(ctx, createdMerchant.ID, page, pageSize)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 0)
	require.Equal(t, int64(0), result.Total)
}

func TestWalletService_RequestPayout_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	blnkClient.GetBalanceFunc = func(ctx context.Context, balanceID string) (int64, error) {
		return 500000, nil // 5000.00 ETB
	}

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		WithBlnkBalanceID("bln_test123").
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	input := domain.RequestPayoutInput{
		AmountCents: 100000, // 1000.00 ETB
		Method:      domain.PayoutMethodTelebirr,
		Destination: "+251911111111",
	}

	payout, err := walletService.RequestPayout(ctx, createdMerchant.ID, input)
	require.NoError(t, err)
	require.NotNil(t, payout)
	require.Equal(t, domain.PayoutStatusPending, payout.Status)
	require.Equal(t, int64(100000), payout.AmountCents)
}

func TestWalletService_RequestPayout_NoWallet(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	input := domain.RequestPayoutInput{
		AmountCents: 100000,
		Method:      domain.PayoutMethodTelebirr,
		Destination: "+251911111111",
	}

	_, err = walletService.RequestPayout(ctx, createdMerchant.ID, input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "merchant does not have a wallet")
}

func TestWalletService_RequestPayout_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	blnkClient.GetBalanceFunc = func(ctx context.Context, balanceID string) (int64, error) {
		return 50000, nil // 500.00 ETB
	}

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		WithBlnkBalanceID("bln_test123").
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	input := domain.RequestPayoutInput{
		AmountCents: 100000, // 1000.00 ETB (more than balance)
		Method:      domain.PayoutMethodTelebirr,
		Destination: "+251911111111",
	}

	_, err = walletService.RequestPayout(ctx, createdMerchant.ID, input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient balance")
}

func TestWalletService_RequestPayout_BelowMinimum(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()
	blnkClient.GetBalanceFunc = func(ctx context.Context, balanceID string) (int64, error) {
		return 500000, nil
	}

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().
		WithUserID(createdUser.ID).
		WithBlnkBalanceID("bln_test123").
		Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	input := domain.RequestPayoutInput{
		AmountCents: 50, // Below 100 cents minimum
		Method:      domain.PayoutMethodTelebirr,
		Destination: "+251911111111",
	}

	_, err = walletService.RequestPayout(ctx, createdMerchant.ID, input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "minimum")
}

func TestWalletService_ListPayoutsByMerchant_Success(t *testing.T) {
	ctx := context.Background()

	pgContainer := testutil.SetupPostgres(t)
	defer testutil.CleanupPostgres(t, pgContainer, nil)

	testutil.RunMigrations(t, pgContainer.ConnectionString)
	pool := testutil.CreatePool(t, pgContainer.ConnectionString)
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	merchantRepo := repository.NewMerchantRepository(pool)
	walletRepo := repository.NewWalletTransactionRepository(pool)
	payoutRepo := repository.NewPayoutRepository(pool)

	blnkClient := mocks.NewMockBlnkClient()

	walletService := NewWalletService(
		merchantRepo,
		walletRepo,
		payoutRepo,
		blnkClient,
	)

	user := testutil.NewUserBuilder().Build()
	createdUser, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	merchant := testutil.NewMerchantBuilder().WithUserID(createdUser.ID).Build()
	createdMerchant, err := merchantRepo.Create(ctx, merchant)
	require.NoError(t, err)

	result, err := walletService.ListPayoutsByMerchant(ctx, createdMerchant.ID, 1, 10)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Items, 0)
	require.Equal(t, int64(0), result.Total)
}
