package mocks

import (
	"context"
	"sync"

	"github.com/SukGebeya/Backend/internal/blnk"
	"github.com/google/uuid"
)

type MockBlnkClient struct {
	CreateMerchantWalletFunc  func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error)
	CreateEscrowHoldFunc      func(ctx context.Context, req blnk.EscrowHoldRequest) (string, error)
	CreditMerchantFunc        func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error)
	CreditPlatformFeeFunc     func(ctx context.Context, req blnk.PlatformFeeRequest) (string, error)
	HoldPayoutFunc            func(ctx context.Context, req blnk.PayoutRequest) (string, error)
	ChargeProcessingFeeFunc   func(ctx context.Context, req blnk.ProcessingFeeRequest) (string, error)
	DebitReversalFunc         func(ctx context.Context, req blnk.ReversalRequest) (string, error)
	CommitInflightFunc        func(ctx context.Context, inflightTxnID string) error
	VoidInflightFunc          func(ctx context.Context, inflightTxnID string) error
	GetBalanceFunc            func(ctx context.Context, balanceID string) (*blnk.Balance, error)
	GetTransactionHistoryFunc func(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error)
	PingFunc                  func(ctx context.Context) error

	mu    sync.Mutex
	Calls map[string]int
}

var _ blnk.BlnkClient = (*MockBlnkClient)(nil)

func NewMockBlnkClient() *MockBlnkClient {
	return &MockBlnkClient{
		CreateMerchantWalletFunc: func(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
			return "bln_" + uuid.New().String()[:8], nil
		},
		CreateEscrowHoldFunc: func(ctx context.Context, req blnk.EscrowHoldRequest) (string, error) {
			return "inflight_" + uuid.New().String()[:8], nil
		},
		CreditMerchantFunc: func(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
			return "txn_" + uuid.New().String()[:8], nil
		},
		CreditPlatformFeeFunc: func(ctx context.Context, req blnk.PlatformFeeRequest) (string, error) {
			return "txn_fee_" + uuid.New().String()[:8], nil
		},
		HoldPayoutFunc: func(ctx context.Context, req blnk.PayoutRequest) (string, error) {
			return "inflight_payout_" + uuid.New().String()[:8], nil
		},
		ChargeProcessingFeeFunc: func(ctx context.Context, req blnk.ProcessingFeeRequest) (string, error) {
			return "txn_chapa_fee_" + uuid.New().String()[:8], nil
		},
		DebitReversalFunc: func(ctx context.Context, req blnk.ReversalRequest) (string, error) {
			return "txn_reversal_" + uuid.New().String()[:8], nil
		},
		CommitInflightFunc: func(ctx context.Context, inflightTxnID string) error {
			return nil
		},
		VoidInflightFunc: func(ctx context.Context, inflightTxnID string) error {
			return nil
		},
		GetBalanceFunc: func(ctx context.Context, balanceID string) (*blnk.Balance, error) {
			return &blnk.Balance{BalanceCents: 100000, AvailableCents: 100000}, nil
		},
		GetTransactionHistoryFunc: func(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error) {
			return []blnk.TransactionResponse{}, nil
		},
		PingFunc: func(ctx context.Context) error {
			return nil
		},
		Calls: make(map[string]int),
	}
}

func (m *MockBlnkClient) record(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Calls == nil {
		m.Calls = make(map[string]int)
	}
	m.Calls[name]++
}

func (m *MockBlnkClient) CallCount(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Calls[name]
}

func (m *MockBlnkClient) CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
	m.record("CreateMerchantWallet")
	return m.CreateMerchantWalletFunc(ctx, merchantID, shopName)
}

func (m *MockBlnkClient) CreateEscrowHold(ctx context.Context, req blnk.EscrowHoldRequest) (string, error) {
	m.record("CreateEscrowHold")
	return m.CreateEscrowHoldFunc(ctx, req)
}

func (m *MockBlnkClient) CreditMerchant(ctx context.Context, req blnk.MerchantCreditRequest) (string, error) {
	m.record("CreditMerchant")
	return m.CreditMerchantFunc(ctx, req)
}

func (m *MockBlnkClient) CreditPlatformFee(ctx context.Context, req blnk.PlatformFeeRequest) (string, error) {
	m.record("CreditPlatformFee")
	return m.CreditPlatformFeeFunc(ctx, req)
}

func (m *MockBlnkClient) HoldPayout(ctx context.Context, req blnk.PayoutRequest) (string, error) {
	m.record("HoldPayout")
	return m.HoldPayoutFunc(ctx, req)
}

func (m *MockBlnkClient) ChargeProcessingFee(ctx context.Context, req blnk.ProcessingFeeRequest) (string, error) {
	m.record("ChargeProcessingFee")
	return m.ChargeProcessingFeeFunc(ctx, req)
}

func (m *MockBlnkClient) DebitMerchantForReversal(ctx context.Context, req blnk.ReversalRequest) (string, error) {
	m.record("DebitMerchantForReversal")
	return m.DebitReversalFunc(ctx, req)
}

func (m *MockBlnkClient) CommitInflight(ctx context.Context, inflightTxnID string) error {
	m.record("CommitInflight")
	return m.CommitInflightFunc(ctx, inflightTxnID)
}

func (m *MockBlnkClient) VoidInflight(ctx context.Context, inflightTxnID string) error {
	m.record("VoidInflight")
	return m.VoidInflightFunc(ctx, inflightTxnID)
}

func (m *MockBlnkClient) GetBalance(ctx context.Context, balanceID string) (*blnk.Balance, error) {
	m.record("GetBalance")
	return m.GetBalanceFunc(ctx, balanceID)
}

func (m *MockBlnkClient) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]blnk.TransactionResponse, error) {
	m.record("GetTransactionHistory")
	return m.GetTransactionHistoryFunc(ctx, balanceID, limit)
}

func (m *MockBlnkClient) Ping(ctx context.Context) error {
	m.record("Ping")
	return m.PingFunc(ctx)
}

func (m *MockBlnkClient) LedgerID() string {
	return "ldg_test"
}

func (m *MockBlnkClient) EscrowBalanceID() string {
	return "bln_escrow_test"
}
