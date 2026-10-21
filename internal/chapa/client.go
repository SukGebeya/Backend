package chapa

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusPending = "pending"

	ModeLive = "live"
	ModeTest = "test"
)

type ChapaClient interface {
	InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error)
	VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error)
	Refund(ctx context.Context, txRef string, req RefundRequest) (*RefundResponse, error)
	InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error)
	VerifyTransfer(ctx context.Context, reference string) (*VerifyTransferResponse, error)
	ListBanks(ctx context.Context) ([]Bank, error)
	VerifyWebhookSignature(payload []byte, signatures ...string) bool
	VerifyPayloadSignature(payload []byte, signature string) bool
}

var _ ChapaClient = (*Client)(nil)

type Client struct {
	secretKey     string
	webhookSecret string
	baseURL       string
	callbackURL   string
	returnURL     string
	httpClient    *http.Client
}

type Config struct {
	BaseURL       string
	SecretKey     string
	WebhookSecret string
	CallbackURL   string
	ReturnURL     string
}

func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.chapa.co/v1"
	}
	return &Client{
		secretKey:     cfg.SecretKey,
		webhookSecret: cfg.WebhookSecret,
		baseURL:       strings.TrimRight(baseURL, "/"),
		callbackURL:   cfg.CallbackURL,
		returnURL:     cfg.ReturnURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type Customization struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type InitializePaymentRequest struct {
	Amount        string         `json:"amount"`
	Currency      string         `json:"currency"`
	Email         string         `json:"email,omitempty"`
	FirstName     string         `json:"first_name,omitempty"`
	LastName      string         `json:"last_name,omitempty"`
	Phone         string         `json:"phone_number,omitempty"`
	TxRef         string         `json:"tx_ref"`
	CallbackURL   string         `json:"callback_url,omitempty"`
	ReturnURL     string         `json:"return_url,omitempty"`
	Customization *Customization `json:"customization,omitempty"`
}

type InitializePaymentData struct {
	CheckoutURL string `json:"checkout_url"`
	TxRef       string `json:"tx_ref"`
}

type InitializePaymentResponse struct {
	Message string                `json:"message"`
	Status  string                `json:"status"`
	Data    InitializePaymentData `json:"data"`
}

type VerifyPaymentData struct {
	TxRef     string      `json:"tx_ref"`
	Reference string      `json:"reference"`
	Amount    json.Number `json:"amount"`
	Charge    json.Number `json:"charge"`
	Currency  string      `json:"currency"`
	Status    string      `json:"status"`
	Mode      string      `json:"mode"`
}

type VerifyPaymentResponse struct {
	Message string            `json:"message"`
	Status  string            `json:"status"`
	Data    VerifyPaymentData `json:"data"`
}

type RefundRequest struct {
	Reason      string `json:"reason,omitempty"`
	AmountCents int64  `json:"-"`
	Reference   string `json:"reference,omitempty"`
}

type RefundResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

type TransferRequest struct {
	AccountName   string
	AccountNumber string
	AmountCents   int64
	Currency      string
	Reference     string
	BankCode      string
}

type TransferResponse struct {
	Message string          `json:"message"`
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data"`
}

type VerifyTransferData struct {
	Reference string      `json:"tx_ref"`
	Amount    json.Number `json:"amount"`
	Status    string      `json:"status"`
}

type VerifyTransferResponse struct {
	Message string             `json:"message"`
	Status  string             `json:"status"`
	Data    VerifyTransferData `json:"data"`
}

type Bank struct {
	ID       json.Number `json:"id"`
	Slug     string      `json:"slug"`
	Name     string      `json:"name"`
	Currency string      `json:"currency"`
	IsMobile int         `json:"is_mobilemoney"`
}

// WebhookPayload accepts both the flat body Chapa sends and the older nested "data" shape.
type WebhookPayload struct {
	Event    string      `json:"event"`
	TxRef    string      `json:"tx_ref"`
	Status   string      `json:"status"`
	Amount   json.Number `json:"amount"`
	Charge   json.Number `json:"charge"`
	Currency string      `json:"currency"`
	Mode     string      `json:"mode"`
	Data     *struct {
		TxRef  string      `json:"tx_ref"`
		Status string      `json:"status"`
		Amount json.Number `json:"amount"`
	} `json:"data,omitempty"`
}

func (p *WebhookPayload) Reference() string {
	if p.TxRef != "" {
		return p.TxRef
	}
	if p.Data != nil {
		return p.Data.TxRef
	}
	return ""
}

func (p *WebhookPayload) PaymentStatus() string {
	if p.Status != "" {
		return p.Status
	}
	if p.Data != nil {
		return p.Data.Status
	}
	return ""
}

func FormatAmount(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// ParseAmountCents converts a decimal ETB string such as "2650.5" into santim without float rounding.
func ParseAmountCents(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("empty amount")
	}
	whole, frac, _ := strings.Cut(amount, ".")
	if len(frac) > 2 {
		if strings.Trim(frac[2:], "0") != "" {
			return 0, fmt.Errorf("amount %q has more than 2 decimal places", amount)
		}
		frac = frac[:2]
	}
	for len(frac) < 2 {
		frac += "0"
	}
	var cents int64
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid amount %q", amount)
		}
		cents = cents*10 + int64(r-'0')
	}
	return cents, nil
}

func (c *Client) InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error) {
	if req.CallbackURL == "" {
		req.CallbackURL = c.callbackURL
	}
	if req.ReturnURL == "" {
		req.ReturnURL = c.returnURL
	}

	var result InitializePaymentResponse
	if err := c.do(ctx, http.MethodPost, "/transaction/initialize", req, &result); err != nil {
		return nil, fmt.Errorf("chapa InitializePayment: %w", err)
	}
	if result.Status != StatusSuccess {
		return nil, fmt.Errorf("chapa InitializePayment returned status %s: %s", result.Status, result.Message)
	}
	return &result, nil
}

func (c *Client) VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error) {
	var result VerifyPaymentResponse
	if err := c.do(ctx, http.MethodGet, "/transaction/verify/"+url.PathEscape(txRef), nil, &result); err != nil {
		return nil, fmt.Errorf("chapa VerifyPayment: %w", err)
	}
	return &result, nil
}

func (c *Client) Refund(ctx context.Context, txRef string, req RefundRequest) (*RefundResponse, error) {
	body := map[string]string{}
	if req.Reason != "" {
		body["reason"] = req.Reason
	}
	if req.Reference != "" {
		body["reference"] = req.Reference
	}
	if req.AmountCents > 0 {
		body["amount"] = FormatAmount(req.AmountCents)
	}

	var result RefundResponse
	if err := c.do(ctx, http.MethodPost, "/refund/"+url.PathEscape(txRef), body, &result); err != nil {
		return nil, fmt.Errorf("chapa Refund: %w", err)
	}
	if result.Status != StatusSuccess {
		return nil, fmt.Errorf("chapa Refund returned status %s: %s", result.Status, result.Message)
	}
	return &result, nil
}

func (c *Client) InitiateTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	body := map[string]string{
		"account_name":   req.AccountName,
		"account_number": req.AccountNumber,
		"amount":         FormatAmount(req.AmountCents),
		"currency":       req.Currency,
		"reference":      req.Reference,
		"bank_code":      req.BankCode,
	}

	var result TransferResponse
	if err := c.do(ctx, http.MethodPost, "/transfers", body, &result); err != nil {
		return nil, fmt.Errorf("chapa InitiateTransfer: %w", err)
	}
	if result.Status != StatusSuccess {
		return nil, fmt.Errorf("chapa InitiateTransfer returned status %s: %s", result.Status, result.Message)
	}
	return &result, nil
}

func (c *Client) VerifyTransfer(ctx context.Context, reference string) (*VerifyTransferResponse, error) {
	var result VerifyTransferResponse
	if err := c.do(ctx, http.MethodGet, "/transfers/verify/"+url.PathEscape(reference), nil, &result); err != nil {
		return nil, fmt.Errorf("chapa VerifyTransfer: %w", err)
	}
	return &result, nil
}

func (c *Client) ListBanks(ctx context.Context) ([]Bank, error) {
	var result struct {
		Message string `json:"message"`
		Data    []Bank `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/banks", nil, &result); err != nil {
		return nil, fmt.Errorf("chapa ListBanks: %w", err)
	}
	return result.Data, nil
}

// VerifyWebhookSignature accepts the x-chapa-signature (HMAC of the payload) or the
// Chapa-Signature (HMAC of the secret itself) header. Payment state is always
// re-verified through the API, so the signature only filters out noise.
func (c *Client) VerifyWebhookSignature(payload []byte, signatures ...string) bool {
	return VerifySignature(c.webhookSecret, payload, signatures...)
}

// VerifyPayloadSignature accepts only x-chapa-signature, the HMAC of the exact
// payload. Events that move money without an API re-check (reversals) require it.
func (c *Client) VerifyPayloadSignature(payload []byte, signature string) bool {
	return VerifyPayload(c.webhookSecret, payload, signature)
}

func VerifyPayload(secret string, payload []byte, signature string) bool {
	if secret == "" || strings.TrimSpace(signature) == "" {
		return false
	}
	return hmac.Equal([]byte(strings.TrimSpace(signature)), []byte(Sign(secret, payload)))
}

func VerifySignature(secret string, payload []byte, signatures ...string) bool {
	if secret == "" {
		return false
	}
	payloadSig := Sign(secret, payload)
	secretSig := Sign(secret, []byte(secret))
	for _, sig := range signatures {
		sig = strings.TrimSpace(sig)
		if sig == "" {
			continue
		}
		if hmac.Equal([]byte(sig), []byte(payloadSig)) || hmac.Equal([]byte(sig), []byte(secretSig)) {
			return true
		}
	}
	return false
}

func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	slog.Debug("chapa: request", "method", method, "path", path)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("chapa: request failed", "method", method, "path", path, "error", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	slog.Debug("chapa: response", "method", method, "path", path, "status_code", resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("chapa: API returned error", "method", method, "path", path, "status_code", resp.StatusCode, "response", string(respBody))
		return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("chapa API returned status %d: %s", e.StatusCode, e.Body)
}
