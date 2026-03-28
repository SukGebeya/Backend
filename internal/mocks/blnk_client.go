package mocks

import (
	"context"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/google/uuid"
)

type MockBlnkClient struct {
	CreateMerchantWalletFunc   func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error)
	CreateEscrowHoldFunc       func(ctx context.Context, req blnk.EscrowHoldRequest) (string, error)
	CommitEscrowFunc           func(ctx context.Context, inflightTxnID string) error
	VoidEscrowFunc             func(ctx context.Context, inflightTxnID string) error
	CreditMerchantFunc         func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error)
	CreditPlatformFeeFunc      func(ctx context.Context, req blnk.PlatformFeeRequest) (string, error)
	DebitMerchantForPayoutFunc func(ctx context.Context, req blnk.PayoutRequest) (string, error)
	GetBalanceFunc             func(ctx context.Context, balanceID string) (int64, error)
	GetTransactionHistoryFunc  func(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error)
}

func NewMockBlnkClient() *MockBlnkClient {
	return &MockBlnkClient{
		CreateMerchantWalletFunc: func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
			return "bln_" + uuid.New().String()[:8], nil
		},
		CreateEscrowHoldFunc: func(ctx context.Context, req blnk.EscrowHoldRequest) (string, error) {
			return "inflight_" + uuid.New().String()[:8], nil
		},
		CommitEscrowFunc: func(ctx context.Context, inflightTxnID string) error {
			return nil
		},
		VoidEscrowFunc: func(ctx context.Context, inflightTxnID string) error {
			return nil
		},
		CreditMerchantFunc: func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
			return "txn_" + uuid.New().String()[:8], nil
		},
		CreditPlatformFeeFunc: func(ctx context.Context, req blnk.PlatformFeeRequest) (string, error) {
			return "txn_fee_" + uuid.New().String()[:8], nil
		},
		DebitMerchantForPayoutFunc: func(ctx context.Context, req blnk.PayoutRequest) (string, error) {
			return "txn_" + uuid.New().String()[:8], nil
		},
		GetBalanceFunc: func(ctx context.Context, balanceID string) (int64, error) {
			return 100000, nil
		},
		GetTransactionHistoryFunc: func(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error) {
			return []blnk.TransactionResponse{}, nil
		},
	}
}

func (m *MockBlnkClient) CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
	return m.CreateMerchantWalletFunc(ctx, merchantID, shopName)
}

func (m *MockBlnkClient) CreateEscrowHold(ctx context.Context, req blnk.EscrowHoldRequest) (string, error) {
	return m.CreateEscrowHoldFunc(ctx, req)
}

func (m *MockBlnkClient) CommitEscrow(ctx context.Context, inflightTxnID string) error {
	return m.CommitEscrowFunc(ctx, inflightTxnID)
}

func (m *MockBlnkClient) VoidEscrow(ctx context.Context, inflightTxnID string) error {
	return m.VoidEscrowFunc(ctx, inflightTxnID)
}

func (m *MockBlnkClient) CreditMerchant(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
	return m.CreditMerchantFunc(ctx, req)
}

func (m *MockBlnkClient) CreditPlatformFee(ctx context.Context, req blnk.PlatformFeeRequest) (string, error) {
	return m.CreditPlatformFeeFunc(ctx, req)
}

func (m *MockBlnkClient) DebitMerchantForPayout(ctx context.Context, req blnk.PayoutRequest) (string, error) {
	return m.DebitMerchantForPayoutFunc(ctx, req)
}

func (m *MockBlnkClient) GetBalance(ctx context.Context, balanceID string) (int64, error) {
	return m.GetBalanceFunc(ctx, balanceID)
}

func (m *MockBlnkClient) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error) {
	return m.GetTransactionHistoryFunc(ctx, balanceID, limit)
}
