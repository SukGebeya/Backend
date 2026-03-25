package blnk

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	blnkgo "github.com/blnkfinance/blnk-go"
	"github.com/google/uuid"
)

type BlnkClient interface {
	CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error)
	CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error)
	CommitEscrow(ctx context.Context, inflightTxnID string) error
	VoidEscrow(ctx context.Context, inflightTxnID string) error
	CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error)
	DebitMerchantForPayout(ctx context.Context, req PayoutRequest) (string, error)
	GetBalance(ctx context.Context, balanceID string) (int64, error)
	GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error)
}

var _ BlnkClient = (*Client)(nil)

type Client struct {
	sdk *blnkgo.Client
}

type Config struct {
	BaseURL          string
	SecretKey        string
	ETBLedgerID      string
	EscrowBalanceID  string
	RevenueBalanceID string
}

func NewClient(cfg Config) (*Client, error) {
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Blnk base URL: %w", err)
	}

	apiKey := cfg.SecretKey
	sdk := blnkgo.NewClient(parsedURL, &apiKey,
		blnkgo.WithTimeout(30e9),
		blnkgo.WithRetry(3),
		blnkgo.WithLogger(blnkgo.NewDefaultLogger()),
	)

	return &Client{sdk: sdk}, nil
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
	ledgerID := os.Getenv("BLNK_ETB_LEDGER_ID")
	if ledgerID == "" {
		slog.Error("CreateMerchantWallet: BLNK_ETB_LEDGER_ID not set")
		return "", fmt.Errorf("BLNK_ETB_LEDGER_ID environment variable not set")
	}

	slog.Debug("CreateMerchantWallet: calling Blnk API",
		"merchant_id", merchantID.String(),
		"shop_name", shopName,
		"ledger_id", ledgerID,
	)

	req := blnkgo.CreateLedgerBalanceRequest{
		LedgerID: ledgerID,
		Currency: "ETB",
		MetaData: map[string]interface{}{
			"merchant_id": merchantID.String(),
			"shop_name":   shopName,
			"type":        "merchant_wallet",
		},
	}

	balance, resp, err := c.sdk.LedgerBalance.Create(req)
	if err != nil {
		slog.Error("CreateMerchantWallet: Blnk API call failed",
			"error", err,
			"merchant_id", merchantID.String(),
			"shop_name", shopName,
		)
		return "", fmt.Errorf("blnk CreateMerchantWallet: %w", err)
	}

	slog.Debug("CreateMerchantWallet: Blnk API call succeeded",
		"balance_id", balance.BalanceID,
		"merchant_id", merchantID.String(),
		"response_status", resp.StatusCode,
	)

	return balance.BalanceID, nil
}

func (c *Client) CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error) {
	escrowBalanceID := os.Getenv("BLNK_ESCROW_BALANCE_ID")
	if escrowBalanceID == "" {
		slog.Error("CreateEscrowHold: BLNK_ESCROW_BALANCE_ID not set")
		return "", fmt.Errorf("BLNK_ESCROW_BALANCE_ID environment variable not set")
	}

	slog.Debug("CreateEscrowHold: calling Blnk API",
		"order_id", req.OrderID.String(),
		"merchant_id", req.MerchantID.String(),
		"amount_cents", req.AmountCents,
	)

	amount := float64(req.AmountCents) / 100.0

	txnReq := blnkgo.CreateTransactionRequest{
		ParentTransaction: blnkgo.ParentTransaction{
			Amount:      amount,
			Currency:    "ETB",
			Precision:   100,
			Reference:   req.OrderID.String(),
			Source:      "@World",
			Destination: escrowBalanceID,
			Description: fmt.Sprintf("Escrow for order %s", req.OrderID.String()),
			MetaData: map[string]interface{}{
				"order_id":    req.OrderID.String(),
				"merchant_id": req.MerchantID.String(),
			},
		},
		AllowOverdraft: true,
		Inflight:       true,
	}

	txn, resp, err := c.sdk.Transaction.Create(txnReq)
	if err != nil {
		slog.Error("CreateEscrowHold: Blnk API call failed",
			"error", err,
			"order_id", req.OrderID.String(),
			"merchant_id", req.MerchantID.String(),
		)
		return "", fmt.Errorf("blnk CreateEscrowHold: %w", err)
	}

	slog.Debug("CreateEscrowHold: Blnk API call succeeded",
		"transaction_id", txn.TransactionID,
		"response_status", resp.StatusCode,
	)

	return txn.TransactionID, nil
}

func (c *Client) CommitEscrow(ctx context.Context, inflightTxnID string) error {
	slog.Debug("CommitEscrow: calling Blnk API",
		"inflight_txn_id", inflightTxnID,
	)

	_, resp, err := c.sdk.Transaction.Update(inflightTxnID, blnkgo.UpdateStatus{
		Status: blnkgo.InflightStatusCommit,
	})
	if err != nil {
		slog.Error("CommitEscrow: Blnk API call failed",
			"error", err,
			"inflight_txn_id", inflightTxnID,
		)
		return fmt.Errorf("blnk CommitEscrow: %w", err)
	}

	slog.Debug("CommitEscrow: Blnk API call succeeded",
		"inflight_txn_id", inflightTxnID,
		"response_status", resp.StatusCode,
	)

	return nil
}

func (c *Client) VoidEscrow(ctx context.Context, inflightTxnID string) error {
	slog.Debug("VoidEscrow: calling Blnk API",
		"inflight_txn_id", inflightTxnID,
	)

	_, resp, err := c.sdk.Transaction.Update(inflightTxnID, blnkgo.UpdateStatus{
		Status: blnkgo.InflightStatusVoid,
	})
	if err != nil {
		slog.Error("VoidEscrow: Blnk API call failed",
			"error", err,
			"inflight_txn_id", inflightTxnID,
		)
		return fmt.Errorf("blnk VoidEscrow: %w", err)
	}

	slog.Debug("VoidEscrow: Blnk API call succeeded",
		"inflight_txn_id", inflightTxnID,
		"response_status", resp.StatusCode,
	)

	return nil
}

func (c *Client) CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error) {
	slog.Debug("CreditMerchant: calling Blnk API",
		"order_id", req.OrderID.String(),
		"merchant_id", req.MerchantID.String(),
		"merchant_blnk_id", req.MerchantBlnkID,
		"subtotal_cents", req.SubtotalCents,
		"fee_cents", req.FeeCents,
	)

	netAmountCents := req.SubtotalCents - req.FeeCents
	amount := float64(netAmountCents) / 100.0

	escrowBalanceID := os.Getenv("BLNK_ESCROW_BALANCE_ID")

	txnReq := blnkgo.CreateTransactionRequest{
		ParentTransaction: blnkgo.ParentTransaction{
			Amount:      amount,
			Currency:    "ETB",
			Precision:   100,
			Reference:   fmt.Sprintf("payout-%s", req.OrderID.String()),
			Source:      escrowBalanceID,
			Destination: req.MerchantBlnkID,
			Description: fmt.Sprintf("Sale credit for order %s", req.OrderID.String()),
			MetaData: map[string]interface{}{
				"order_id":    req.OrderID.String(),
				"merchant_id": req.MerchantID.String(),
			},
		},
		AllowOverdraft: false,
		Inflight:       false,
	}

	txn, resp, err := c.sdk.Transaction.Create(txnReq)
	if err != nil {
		slog.Error("CreditMerchant: Blnk API call failed",
			"error", err,
			"order_id", req.OrderID.String(),
			"merchant_id", req.MerchantID.String(),
		)
		return "", fmt.Errorf("blnk CreditMerchant: %w", err)
	}

	slog.Debug("CreditMerchant: Blnk API call succeeded",
		"transaction_id", txn.TransactionID,
		"response_status", resp.StatusCode,
	)

	return txn.TransactionID, nil
}

func (c *Client) DebitMerchantForPayout(ctx context.Context, req PayoutRequest) (string, error) {
	slog.Debug("DebitMerchantForPayout: calling Blnk API",
		"payout_id", req.PayoutID.String(),
		"merchant_id", req.MerchantID.String(),
		"merchant_blnk_id", req.MerchantBlnkID,
		"amount_cents", req.AmountCents,
	)

	amount := float64(req.AmountCents) / 100.0

	txnReq := blnkgo.CreateTransactionRequest{
		ParentTransaction: blnkgo.ParentTransaction{
			Amount:      amount,
			Currency:    "ETB",
			Precision:   100,
			Reference:   req.PayoutID.String(),
			Source:      req.MerchantBlnkID,
			Destination: "@World",
			Description: fmt.Sprintf("Payout to %s", req.Destination),
			MetaData: map[string]interface{}{
				"payout_id":   req.PayoutID.String(),
				"merchant_id": req.MerchantID.String(),
				"method":      req.Method,
				"destination": req.Destination,
			},
		},
		AllowOverdraft: false,
		Inflight:       false,
	}

	txn, resp, err := c.sdk.Transaction.Create(txnReq)
	if err != nil {
		slog.Error("DebitMerchantForPayout: Blnk API call failed",
			"error", err,
			"payout_id", req.PayoutID.String(),
			"merchant_id", req.MerchantID.String(),
		)
		return "", fmt.Errorf("blnk DebitMerchantForPayout: %w", err)
	}

	slog.Debug("DebitMerchantForPayout: Blnk API call succeeded",
		"transaction_id", txn.TransactionID,
		"response_status", resp.StatusCode,
	)

	return txn.TransactionID, nil
}

func (c *Client) GetBalance(ctx context.Context, balanceID string) (int64, error) {
	slog.Debug("GetBalance: calling Blnk API",
		"balance_id", balanceID,
	)

	balance, resp, err := c.sdk.LedgerBalance.Get(balanceID)
	if err != nil {
		slog.Error("GetBalance: Blnk API call failed",
			"error", err,
			"balance_id", balanceID,
		)
		return 0, fmt.Errorf("blnk GetBalance: %w", err)
	}

	slog.Debug("GetBalance: Blnk API call succeeded",
		"balance_id", balanceID,
		"response_status", resp.StatusCode,
	)

	if balance.Balance == nil {
		return 0, nil
	}

	return balance.Balance.Int64() * 100, nil
}

func (c *Client) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error) {
	slog.Debug("GetTransactionHistory: calling Blnk API",
		"balance_id", balanceID,
		"limit", limit,
	)

	filterParams := blnkgo.FilterParams{
		Filters: []blnkgo.Filter{
			{
				Field:    "source",
				Operator: blnkgo.OpEqual,
				Value:    balanceID,
			},
		},
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	filterResp, resp, err := c.sdk.Transaction.Filter(filterParams)
	if err != nil {
		slog.Error("GetTransactionHistory: Blnk API call failed",
			"error", err,
			"balance_id", balanceID,
		)
		return nil, fmt.Errorf("blnk GetTransactionHistory: %w", err)
	}

	slog.Debug("GetTransactionHistory: Blnk API call succeeded",
		"balance_id", balanceID,
		"response_status", resp.StatusCode,
	)

	if filterResp.Data == nil {
		return []TransactionResponse{}, nil
	}

	dataSlice, ok := filterResp.Data.([]interface{})
	if !ok {
		return []TransactionResponse{}, nil
	}

	result := make([]TransactionResponse, 0, len(dataSlice))
	for _, item := range dataSlice {
		txnMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		txnID, _ := txnMap["transaction_id"].(string)
		ref, _ := txnMap["reference"].(string)
		status, _ := txnMap["status"].(string)
		result = append(result, TransactionResponse{
			TransactionID: txnID,
			Reference:     ref,
			Status:        status,
		})
	}

	return result, nil
}
