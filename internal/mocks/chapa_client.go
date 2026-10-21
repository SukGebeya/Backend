package mocks

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/SukGebeya/Backend/internal/chapa"
)

type MockChapaClient struct {
	InitializePaymentFunc      func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error)
	VerifyPaymentFunc          func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error)
	RefundFunc                 func(ctx context.Context, txRef string, req chapa.RefundRequest) (*chapa.RefundResponse, error)
	InitiateTransferFunc       func(ctx context.Context, req chapa.TransferRequest) (*chapa.TransferResponse, error)
	VerifyTransferFunc         func(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error)
	ListBanksFunc              func(ctx context.Context) ([]chapa.Bank, error)
	VerifyWebhookSignatureFunc func(payload []byte, signatures ...string) bool
	VerifyPayloadSignatureFunc func(payload []byte, signature string) bool

	mu    sync.Mutex
	Calls map[string]int
}

var _ chapa.ChapaClient = (*MockChapaClient)(nil)

func NewMockChapaClient() *MockChapaClient {
	return &MockChapaClient{
		InitializePaymentFunc: func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
			return &chapa.InitializePaymentResponse{
				Message: "Hosted Link",
				Status:  chapa.StatusSuccess,
				Data:    chapa.InitializePaymentData{CheckoutURL: "https://checkout.chapa.co/test", TxRef: req.TxRef},
			}, nil
		},
		VerifyPaymentFunc: func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
			return &chapa.VerifyPaymentResponse{
				Message: "Payment details",
				Status:  chapa.StatusSuccess,
				Data:    chapa.VerifyPaymentData{TxRef: txRef, Status: chapa.StatusSuccess, Amount: json.Number("0"), Currency: "ETB"},
			}, nil
		},
		RefundFunc: func(ctx context.Context, txRef string, req chapa.RefundRequest) (*chapa.RefundResponse, error) {
			return &chapa.RefundResponse{Message: "Refund processed successfully", Status: chapa.StatusSuccess}, nil
		},
		InitiateTransferFunc: func(ctx context.Context, req chapa.TransferRequest) (*chapa.TransferResponse, error) {
			return &chapa.TransferResponse{Message: "Transfer Queued Successfully", Status: chapa.StatusSuccess}, nil
		},
		VerifyTransferFunc: func(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error) {
			return &chapa.VerifyTransferResponse{
				Status: chapa.StatusSuccess,
				Data:   chapa.VerifyTransferData{Reference: reference, Status: chapa.StatusSuccess},
			}, nil
		},
		ListBanksFunc: func(ctx context.Context) ([]chapa.Bank, error) {
			return []chapa.Bank{{ID: "855", Slug: "telebirr", Name: "telebirr", Currency: "ETB", IsMobile: 1}}, nil
		},
		VerifyWebhookSignatureFunc: func(payload []byte, signatures ...string) bool {
			return true
		},
		VerifyPayloadSignatureFunc: func(payload []byte, signature string) bool {
			return true
		},
		Calls: make(map[string]int),
	}
}

func (m *MockChapaClient) record(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Calls == nil {
		m.Calls = make(map[string]int)
	}
	m.Calls[name]++
}

func (m *MockChapaClient) CallCount(name string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Calls[name]
}

func (m *MockChapaClient) InitializePayment(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
	m.record("InitializePayment")
	return m.InitializePaymentFunc(ctx, req)
}

func (m *MockChapaClient) VerifyPayment(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
	m.record("VerifyPayment")
	return m.VerifyPaymentFunc(ctx, txRef)
}

func (m *MockChapaClient) Refund(ctx context.Context, txRef string, req chapa.RefundRequest) (*chapa.RefundResponse, error) {
	m.record("Refund")
	return m.RefundFunc(ctx, txRef, req)
}

func (m *MockChapaClient) InitiateTransfer(ctx context.Context, req chapa.TransferRequest) (*chapa.TransferResponse, error) {
	m.record("InitiateTransfer")
	return m.InitiateTransferFunc(ctx, req)
}

func (m *MockChapaClient) VerifyTransfer(ctx context.Context, reference string) (*chapa.VerifyTransferResponse, error) {
	m.record("VerifyTransfer")
	return m.VerifyTransferFunc(ctx, reference)
}

func (m *MockChapaClient) ListBanks(ctx context.Context) ([]chapa.Bank, error) {
	m.record("ListBanks")
	return m.ListBanksFunc(ctx)
}

func (m *MockChapaClient) VerifyWebhookSignature(payload []byte, signatures ...string) bool {
	return m.VerifyWebhookSignatureFunc(payload, signatures...)
}

func (m *MockChapaClient) VerifyPayloadSignature(payload []byte, signature string) bool {
	return m.VerifyPayloadSignatureFunc(payload, signature)
}
