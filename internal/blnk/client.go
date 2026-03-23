package blnk

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Client struct {
	baseURL    string
	secretKey  string
	httpClient HTTPClient
}

type HTTPClient interface {
	Do(req *Request) (*Response, error)
}

type Request struct {
	Method  string
	URL     string
	Body    []byte
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Body       []byte
}

type Config struct {
	BaseURL   string
	SecretKey string
}

func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:   cfg.BaseURL,
		secretKey: cfg.SecretKey,
	}
}

type CreateBalanceRequest struct {
	LedgerID string         `json:"ledger_id"`
	Currency string         `json:"currency"`
	MetaData map[string]any `json:"metadata"`
}

type BalanceResponse struct {
	BalanceID string `json:"balance_id"`
	LedgerID  string `json:"ledger_id"`
	Currency  string `json:"currency"`
}

type CreateTransactionRequest struct {
	Amount         float64        `json:"amount"`
	Currency       string         `json:"currency"`
	Precision      int            `json:"precision"`
	Reference      string         `json:"reference"`
	Source         string         `json:"source"`
	Destination    string         `json:"destination"`
	Description    string         `json:"description"`
	AllowOverdraft bool           `json:"allow_overdraft"`
	Inflight       bool           `json:"inflight"`
	MetaData       map[string]any `json:"metadata"`
}

type TransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
}

type UpdateInflightRequest struct {
	Status string `json:"status"`
}

type EscrowHoldRequest struct {
	OrderID     uuid.UUID
	MerchantID  uuid.UUID
	AmountCents int64
}

type MerchantCreditRequest struct {
	OrderID        uuid.UUID
	MerchantID     uuid.UUID
	SubtotalCents  int64
	FeeCents       int64
	MerchantBlnkID string
}

type PayoutRequest struct {
	PayoutID       uuid.UUID
	MerchantID     uuid.UUID
	AmountCents    int64
	Method         string
	Destination    string
	MerchantBlnkID string
}

func (c *Client) CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
	return "", fmt.Errorf("CreateMerchantWallet: not implemented - requires blnk-go SDK integration")
}

func (c *Client) CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error) {
	return "", fmt.Errorf("CreateEscrowHold: not implemented - requires blnk-go SDK integration")
}

func (c *Client) CommitEscrow(ctx context.Context, inflightTxnID string) error {
	return fmt.Errorf("CommitEscrow: not implemented - requires blnk-go SDK integration")
}

func (c *Client) VoidEscrow(ctx context.Context, inflightTxnID string) error {
	return fmt.Errorf("VoidEscrow: not implemented - requires blnk-go SDK integration")
}

func (c *Client) CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error) {
	return "", fmt.Errorf("CreditMerchant: not implemented - requires blnk-go SDK integration")
}

func (c *Client) DebitMerchantForPayout(ctx context.Context, req PayoutRequest) (string, error) {
	return "", fmt.Errorf("DebitMerchantForPayout: not implemented - requires blnk-go SDK integration")
}

func (c *Client) GetBalance(ctx context.Context, balanceID string) (int64, error) {
	return 0, fmt.Errorf("GetBalance: not implemented - requires blnk-go SDK integration")
}

func (c *Client) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error) {
	return nil, fmt.Errorf("GetTransactionHistory: not implemented - requires blnk-go SDK integration")
}
