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
	"time"
)

type ChapaClient interface {
	InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error)
	VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error)
	VerifyWebhookSignature(payload []byte, signature string) bool
}

var _ ChapaClient = (*Client)(nil)

type Client struct {
	secretKey   string
	baseURL     string
	callbackURL string
	returnURL   string
	httpClient  *http.Client
}

type Config struct {
	SecretKey   string
	CallbackURL string
	ReturnURL   string
}

func NewClient(cfg Config) *Client {
	return &Client{
		secretKey:   cfg.SecretKey,
		baseURL:     "https://api.chapa.co/v1",
		callbackURL: cfg.CallbackURL,
		returnURL:   cfg.ReturnURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type InitializePaymentRequest struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Phone       string `json:"phone_number"`
	TxRef       string `json:"tx_ref"`
	CallbackURL string `json:"callback_url"`
	ReturnURL   string `json:"return_url"`
	Description string `json:"customization[description]"`
}

type InitializePaymentResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
		TxRef       string `json:"tx_ref"`
	} `json:"data"`
}

type VerifyPaymentResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		TxRef   string `json:"tx_ref"`
		Amount  string `json:"amount"`
		Status  string `json:"status"`
		ChapaID string `json:"id"`
	} `json:"data"`
}

type WebhookPayload struct {
	Event string `json:"event"`
	Data  struct {
		TxRef  string `json:"tx_ref"`
		Amount string `json:"amount"`
		Status string `json:"status"`
	} `json:"data"`
}

func (c *Client) InitializePayment(ctx context.Context, req InitializePaymentRequest) (*InitializePaymentResponse, error) {
	slog.Debug("InitializePayment: calling Chapa API",
		"tx_ref", req.TxRef,
		"amount", req.Amount,
		"currency", req.Currency,
	)

	if req.CallbackURL == "" {
		req.CallbackURL = c.callbackURL
	}
	if req.ReturnURL == "" {
		req.ReturnURL = c.returnURL
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/transaction/initialize", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("InitializePayment: Chapa API call failed",
			"error", err,
			"tx_ref", req.TxRef,
		)
		return nil, fmt.Errorf("chapa API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.Debug("InitializePayment: Chapa API response",
		"status_code", resp.StatusCode,
		"tx_ref", req.TxRef,
	)

	if resp.StatusCode != http.StatusOK {
		slog.Error("InitializePayment: Chapa API returned error",
			"status_code", resp.StatusCode,
			"response", string(respBody),
			"tx_ref", req.TxRef,
		)
		return nil, fmt.Errorf("chapa API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result InitializePaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Status != "success" {
		slog.Error("InitializePayment: Chapa returned non-success status",
			"status", result.Status,
			"message", result.Message,
			"tx_ref", req.TxRef,
		)
		return nil, fmt.Errorf("chapa returned status %s: %s", result.Status, result.Message)
	}

	slog.Debug("InitializePayment: Chapa API call succeeded",
		"tx_ref", req.TxRef,
		"checkout_url", result.Data.CheckoutURL,
	)

	return &result, nil
}

func (c *Client) VerifyPayment(ctx context.Context, txRef string) (*VerifyPaymentResponse, error) {
	slog.Debug("VerifyPayment: calling Chapa API",
		"tx_ref", txRef,
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/transaction/verify/"+txRef, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("VerifyPayment: Chapa API call failed",
			"error", err,
			"tx_ref", txRef,
		)
		return nil, fmt.Errorf("chapa API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.Debug("VerifyPayment: Chapa API response",
		"status_code", resp.StatusCode,
		"tx_ref", txRef,
	)

	if resp.StatusCode != http.StatusOK {
		slog.Error("VerifyPayment: Chapa API returned error",
			"status_code", resp.StatusCode,
			"response", string(respBody),
			"tx_ref", txRef,
		)
		return nil, fmt.Errorf("chapa API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result VerifyPaymentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	slog.Debug("VerifyPayment: Chapa API call succeeded",
		"tx_ref", txRef,
		"status", result.Data.Status,
	)

	return &result, nil
}

func (c *Client) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	slog.Debug("VerifyWebhookSignature: verifying signature",
		"expected_len", len(expectedSignature),
		"received_len", len(signature),
	)

	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}
