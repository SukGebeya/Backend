package chapa

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
)

// FakeClient is an in-memory stand-in for Chapa used when CHAPA_MODE=mock.
// Every initialized payment verifies as successful for the exact amount, with a
// 3.5% processing charge like Chapa's, in test mode. Transfers and refunds
// always succeed.
type FakeClient struct {
	webhookSecret string
	returnURL     string

	mu        sync.Mutex
	payments  map[string]int64
	transfers map[string]int64
}

var _ ChapaClient = (*FakeClient)(nil)

func NewFakeClient(cfg Config) *FakeClient {
	return &FakeClient{
		webhookSecret: cfg.WebhookSecret,
		returnURL:     cfg.ReturnURL,
		payments:      make(map[string]int64),
		transfers:     make(map[string]int64),
	}
}

func (f *FakeClient) InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error) {
	cents, err := ParseAmountCents(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("fake chapa: %w", err)
	}

	f.mu.Lock()
	f.payments[req.TxRef] = cents
	f.mu.Unlock()

	checkoutURL := "http://localhost/mock-chapa/checkout?tx_ref=" + url.QueryEscape(req.TxRef)
	if f.returnURL != "" {
		checkoutURL = f.returnURL + "?tx_ref=" + url.QueryEscape(req.TxRef)
	}

	slog.Info("[MOCK CHAPA] payment initialized", "tx_ref", req.TxRef, "amount", req.Amount)

	return &InitializePaymentResponse{
		Message: "Hosted Link",
		Status:  StatusSuccess,
		Data:    InitializePaymentData{CheckoutURL: checkoutURL, TxRef: req.TxRef},
	}, nil
}

func (f *FakeClient) VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error) {
	f.mu.Lock()
	cents, ok := f.payments[txRef]
	f.mu.Unlock()

	if !ok {
		return nil, &APIError{StatusCode: 404, Body: `{"message":"Invalid transaction or Transaction not found","status":"failed"}`}
	}

	return &VerifyPaymentResponse{
		Message: "Payment details",
		Status:  StatusSuccess,
		Data: VerifyPaymentData{
			TxRef:    txRef,
			Amount:   json.Number(FormatAmount(cents)),
			Charge:   json.Number(FormatAmount(FakeChargeCents(cents))),
			Currency: "ETB",
			Status:   StatusSuccess,
			Mode:     ModeTest,
		},
	}, nil
}

func (f *FakeClient) Refund(ctx context.Context, txRef string, req RefundRequest) (*RefundResponse, error) {
	slog.Info("[MOCK CHAPA] refund processed", "tx_ref", txRef, "reason", req.Reason)
	return &RefundResponse{Message: "Refund processed successfully", Status: StatusSuccess}, nil
}

func (f *FakeClient) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	f.mu.Lock()
	f.transfers[req.Reference] = req.AmountCents
	f.mu.Unlock()

	slog.Info("[MOCK CHAPA] transfer queued", "reference", req.Reference, "amount", FormatAmount(req.AmountCents), "bank_code", req.BankCode)

	data, _ := json.Marshal(req.Reference)
	return &TransferResponse{Message: "Transfer Queued Successfully", Status: StatusSuccess, Data: data}, nil
}

func (f *FakeClient) VerifyTransfer(ctx context.Context, reference string) (*VerifyTransferResponse, error) {
	f.mu.Lock()
	cents, ok := f.transfers[reference]
	f.mu.Unlock()

	if !ok {
		return nil, &APIError{StatusCode: 404, Body: `{"message":"Transfer not found","status":"failed"}`}
	}

	return &VerifyTransferResponse{
		Message: "Transfer details",
		Status:  StatusSuccess,
		Data:    VerifyTransferData{Reference: reference, Amount: json.Number(FormatAmount(cents)), Status: StatusSuccess},
	}, nil
}

func (f *FakeClient) ListBanks(ctx context.Context) ([]Bank, error) {
	return []Bank{
		{ID: "855", Slug: "telebirr", Name: "telebirr", Currency: "ETB", IsMobile: 1},
		{ID: "128", Slug: "cbebirr", Name: "CBEBirr", Currency: "ETB", IsMobile: 1},
		{ID: "946", Slug: "cbe", Name: "Commercial Bank of Ethiopia (CBE)", Currency: "ETB"},
	}, nil
}

// FakeChargeCents mirrors Chapa's 3.5% processing charge, rounded to the santim.
func FakeChargeCents(amountCents int64) int64 {
	return (amountCents*35 + 500) / 1000
}

func (f *FakeClient) VerifyPayloadSignature(payload []byte, signature string) bool {
	return VerifyPayload(f.webhookSecret, payload, signature)
}

func (f *FakeClient) VerifyWebhookSignature(payload []byte, signatures ...string) bool {
	return VerifySignature(f.webhookSecret, payload, signatures...)
}
