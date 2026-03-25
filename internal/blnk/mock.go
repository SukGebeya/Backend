package blnk

import (
	"context"

	"github.com/google/uuid"
)

type MockClient struct {
	CreateMerchantWalletFunc   func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error)
	CreateEscrowHoldFunc       func(ctx context.Context, req EscrowHoldRequest) (string, error)
	CommitEscrowFunc           func(ctx context.Context, inflightTxnID string) error
	VoidEscrowFunc             func(ctx context.Context, inflightTxnID string) error
	CreditMerchantFunc         func(ctx context.Context, req MerchantCreditRequest) (string, error)
	DebitMerchantForPayoutFunc func(ctx context.Context, req PayoutRequest) (string, error)
	GetBalanceFunc             func(ctx context.Context, balanceID string) (int64, error)
	GetTransactionHistoryFunc  func(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error)

	CreateMerchantWalletCalled   bool
	CreateMerchantWalletCalls    []CreateMerchantWalletCall
	CreateEscrowHoldCalled       bool
	CommitEscrowCalled           bool
	VoidEscrowCalled             bool
	CreditMerchantCalled         bool
	DebitMerchantForPayoutCalled bool
	GetBalanceCalled             bool
	GetTransactionHistoryCalled  bool
}

type CreateMerchantWalletCall struct {
	MerchantID uuid.UUID
	ShopName   string
}

func (m *MockClient) CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
	m.CreateMerchantWalletCalled = true
	m.CreateMerchantWalletCalls = append(m.CreateMerchantWalletCalls, CreateMerchantWalletCall{
		MerchantID: merchantID,
		ShopName:   shopName,
	})
	if m.CreateMerchantWalletFunc != nil {
		return m.CreateMerchantWalletFunc(ctx, merchantID, shopName)
	}
	return "blnk_balance_" + merchantID.String()[:8], nil
}

func (m *MockClient) CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error) {
	m.CreateEscrowHoldCalled = true
	if m.CreateEscrowHoldFunc != nil {
		return m.CreateEscrowHoldFunc(ctx, req)
	}
	return "txn_" + req.OrderID.String()[:8], nil
}

func (m *MockClient) CommitEscrow(ctx context.Context, inflightTxnID string) error {
	m.CommitEscrowCalled = true
	if m.CommitEscrowFunc != nil {
		return m.CommitEscrowFunc(ctx, inflightTxnID)
	}
	return nil
}

func (m *MockClient) VoidEscrow(ctx context.Context, inflightTxnID string) error {
	m.VoidEscrowCalled = true
	if m.VoidEscrowFunc != nil {
		return m.VoidEscrowFunc(ctx, inflightTxnID)
	}
	return nil
}

func (m *MockClient) CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error) {
	m.CreditMerchantCalled = true
	if m.CreditMerchantFunc != nil {
		return m.CreditMerchantFunc(ctx, req)
	}
	return "txn_credit_" + req.OrderID.String()[:8], nil
}

func (m *MockClient) DebitMerchantForPayout(ctx context.Context, req PayoutRequest) (string, error) {
	m.DebitMerchantForPayoutCalled = true
	if m.DebitMerchantForPayoutFunc != nil {
		return m.DebitMerchantForPayoutFunc(ctx, req)
	}
	return "txn_payout_" + req.PayoutID.String()[:8], nil
}

func (m *MockClient) GetBalance(ctx context.Context, balanceID string) (int64, error) {
	m.GetBalanceCalled = true
	if m.GetBalanceFunc != nil {
		return m.GetBalanceFunc(ctx, balanceID)
	}
	return 0, nil
}

func (m *MockClient) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error) {
	m.GetTransactionHistoryCalled = true
	if m.GetTransactionHistoryFunc != nil {
		return m.GetTransactionHistoryFunc(ctx, balanceID, limit)
	}
	return []TransactionResponse{}, nil
}
