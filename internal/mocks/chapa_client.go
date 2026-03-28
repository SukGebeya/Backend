package mocks

import (
	"context"

	"github.com/SukGebeya/Backend/internal/chapa"
)

type MockChapaClient struct {
	InitializePaymentFunc      func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error)
	VerifyPaymentFunc          func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error)
	VerifyWebhookSignatureFunc func(payload []byte, signature string) bool
}

func NewMockChapaClient() *MockChapaClient {
	return &MockChapaClient{
		InitializePaymentFunc: func(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
			return &chapa.InitializePaymentResponse{
				Message: "Payment initialized successfully",
				Status:  "success",
				Data: struct {
					CheckoutURL string `json:"checkout_url"`
					TxRef       string `json:"tx_ref"`
				}{
					CheckoutURL: "https://checkout.chapa.co/test",
					TxRef:       req.TxRef,
				},
			}, nil
		},
		VerifyPaymentFunc: func(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
			return &chapa.VerifyPaymentResponse{
				Message: "Payment verified",
				Status:  "success",
			}, nil
		},
		VerifyWebhookSignatureFunc: func(payload []byte, signature string) bool {
			return true
		},
	}
}

func (m *MockChapaClient) InitializePayment(ctx context.Context, req chapa.InitializePaymentRequest) (*chapa.InitializePaymentResponse, error) {
	return m.InitializePaymentFunc(ctx, req)
}

func (m *MockChapaClient) VerifyPayment(ctx context.Context, txRef string) (*chapa.VerifyPaymentResponse, error) {
	return m.VerifyPaymentFunc(ctx, txRef)
}

func (m *MockChapaClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	return m.VerifyWebhookSignatureFunc(payload, signature)
}
