package chapa

import (
	"context"
)

type MockClient struct {
	InitializePaymentFunc      func(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error)
	VerifyPaymentFunc          func(ctx context.Context, txRef string) (*VerifyPaymentResponse, error)
	VerifyWebhookSignatureFunc func(payload []byte, signature string) bool
}

var _ ChapaClient = (*MockClient)(nil)

func (m *MockClient) InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error) {
	if m.InitializePaymentFunc != nil {
		return m.InitializePaymentFunc(ctx, req)
	}
	return &InitializePaymentResponse{
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
}

func (m *MockClient) VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error) {
	if m.VerifyPaymentFunc != nil {
		return m.VerifyPaymentFunc(ctx, txRef)
	}
	return &VerifyPaymentResponse{
		Message: "Payment verified successfully",
		Status:  "success",
		Data: struct {
			TxRef   string `json:"tx_ref"`
			Amount  string `json:"amount"`
			Status  string `json:"status"`
			ChapaID string `json:"id"`
		}{
			TxRef:   txRef,
			Amount:  "1000.00",
			Status:  "success",
			ChapaID: "test-chapa-id",
		},
	}, nil
}

func (m *MockClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	if m.VerifyWebhookSignatureFunc != nil {
		return m.VerifyWebhookSignatureFunc(payload, signature)
	}
	return true
}
