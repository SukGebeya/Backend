package blnk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"time"

	blnkgo "github.com/blnkfinance/blnk-go"
	"github.com/google/uuid"
)

const (
	currencyETB    = "ETB"
	precision      = 100
	worldSource    = "@World"
	metaType       = "type"
	metaOrderID    = "order_id"
	metaMerchantID = "merchant_id"
)

type BlnkClient interface {
	CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error)
	CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error)
	CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error)
	CreditPlatformFee(ctx context.Context, req PlatformFeeRequest) (string, error)
	HoldPayout(ctx context.Context, req PayoutRequest) (string, error)
	ChargeProcessingFee(ctx context.Context, req ProcessingFeeRequest) (string, error)
	DebitMerchantForReversal(ctx context.Context, req ReversalRequest) (string, error)
	CommitInflight(ctx context.Context, inflightTxnID string) error
	VoidInflight(ctx context.Context, inflightTxnID string) error
	GetBalance(ctx context.Context, balanceID string) (*Balance, error)
	GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error)
	Ping(ctx context.Context) error
	LedgerID() string
	EscrowBalanceID() string
}

var _ BlnkClient = (*Client)(nil)

type Client struct {
	sdk                    *blnkgo.Client
	ledgerID               string
	escrowBalanceID        string
	revenueBalanceID       string
	processingFeeBalanceID string
}

type Config struct {
	BaseURL                string
	SecretKey              string
	ETBLedgerID            string
	EscrowBalanceID        string
	RevenueBalanceID       string
	ProcessingFeeBalanceID string
}

// ErrInflightBusy means a commit or void for the transaction is already queued in
// Blnk; callers should retry shortly.
var ErrInflightBusy = errors.New("blnk inflight action already queued")

func NewClient(cfg Config) (*Client, error) {
	parsedURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Blnk base URL: %w", err)
	}

	var apiKey *string
	if cfg.SecretKey != "" {
		apiKey = &cfg.SecretKey
	}

	sdk := blnkgo.NewClient(parsedURL, apiKey,
		blnkgo.WithTimeout(30*time.Second),
		blnkgo.WithRetry(3),
	)

	return &Client{
		sdk:                    sdk,
		ledgerID:               cfg.ETBLedgerID,
		escrowBalanceID:        cfg.EscrowBalanceID,
		revenueBalanceID:       cfg.RevenueBalanceID,
		processingFeeBalanceID: cfg.ProcessingFeeBalanceID,
	}, nil
}

type Balance struct {
	BalanceCents        int64
	InflightDebitCents  int64
	InflightCreditCents int64
	AvailableCents      int64
}

type TransactionResponse struct {
	TransactionID string `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
}

type EscrowHoldRequest struct {
	OrderID     uuid.UUID
	MerchantID  uuid.UUID
	AmountCents int64
}

type MerchantCreditRequest struct {
	OrderID          uuid.UUID
	MerchantID       uuid.UUID
	SubtotalCents    int64
	FeeCents         int64
	DeliveryFeeCents int64
	MerchantBlnkID   string
}

type PlatformFeeRequest struct {
	OrderID  uuid.UUID
	FeeCents int64
}

type ProcessingFeeRequest struct {
	OrderID     uuid.UUID
	AmountCents int64
}

type ReversalRequest struct {
	OrderID        uuid.UUID
	MerchantID     uuid.UUID
	AmountCents    int64
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

func EscrowReference(orderID uuid.UUID) string {
	return "escrow-" + orderID.String()
}

func SaleCreditReference(orderID uuid.UUID) string {
	return "sale-credit-" + orderID.String()
}

func PlatformFeeReference(orderID uuid.UUID) string {
	return "platform-fee-" + orderID.String()
}

func ProcessingFeeReference(orderID uuid.UUID) string {
	return "chapa-fee-" + orderID.String()
}

func ReversalReference(orderID uuid.UUID) string {
	return "reversal-" + orderID.String()
}

func PayoutReference(payoutID uuid.UUID) string {
	return "payout-" + payoutID.String()
}

func (c *Client) LedgerID() string {
	return c.ledgerID
}

func (c *Client) EscrowBalanceID() string {
	return c.escrowBalanceID
}

func (c *Client) CreateMerchantWallet(ctx context.Context, merchantID uuid.UUID, shopName string) (string, error) {
	if c.ledgerID == "" {
		return "", fmt.Errorf("blnk CreateMerchantWallet: BLNK_ETB_LEDGER_ID is not configured")
	}

	balance, _, err := c.sdk.LedgerBalance.Create(blnkgo.CreateLedgerBalanceRequest{
		LedgerID: c.ledgerID,
		Currency: currencyETB,
		MetaData: map[string]interface{}{
			metaMerchantID: merchantID.String(),
			"shop_name":    shopName,
			metaType:       "merchant_wallet",
		},
	})
	if err != nil {
		slog.Error("blnk: CreateMerchantWallet failed", "merchant_id", merchantID, "error", err)
		return "", fmt.Errorf("blnk CreateMerchantWallet: %w", err)
	}

	slog.Info("blnk: merchant wallet created", "merchant_id", merchantID, "balance_id", balance.BalanceID)
	return balance.BalanceID, nil
}

func (c *Client) CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (string, error) {
	if c.escrowBalanceID == "" {
		return "", fmt.Errorf("blnk CreateEscrowHold: BLNK_ESCROW_BALANCE_ID is not configured")
	}
	return c.createTransaction("CreateEscrowHold", txnSpec{
		reference:      EscrowReference(req.OrderID),
		amountCents:    req.AmountCents,
		source:         worldSource,
		destination:    c.escrowBalanceID,
		description:    fmt.Sprintf("Escrow for order %s", req.OrderID),
		inflight:       true,
		allowOverdraft: true,
		meta: map[string]interface{}{
			metaOrderID:    req.OrderID.String(),
			metaMerchantID: req.MerchantID.String(),
			metaType:       "escrow_hold",
		},
	})
}

func (c *Client) CreditMerchant(ctx context.Context, req MerchantCreditRequest) (string, error) {
	if c.escrowBalanceID == "" {
		return "", fmt.Errorf("blnk CreditMerchant: BLNK_ESCROW_BALANCE_ID is not configured")
	}
	return c.createTransaction("CreditMerchant", txnSpec{
		reference:   SaleCreditReference(req.OrderID),
		amountCents: req.SubtotalCents - req.FeeCents + req.DeliveryFeeCents,
		source:      c.escrowBalanceID,
		destination: req.MerchantBlnkID,
		description: fmt.Sprintf("Sale credit for order %s", req.OrderID),
		meta: map[string]interface{}{
			metaOrderID:    req.OrderID.String(),
			metaMerchantID: req.MerchantID.String(),
			metaType:       "sale_credit",
		},
	})
}

func (c *Client) CreditPlatformFee(ctx context.Context, req PlatformFeeRequest) (string, error) {
	if c.escrowBalanceID == "" || c.revenueBalanceID == "" {
		return "", fmt.Errorf("blnk CreditPlatformFee: escrow and revenue balances must be configured")
	}
	return c.createTransaction("CreditPlatformFee", txnSpec{
		reference:   PlatformFeeReference(req.OrderID),
		amountCents: req.FeeCents,
		source:      c.escrowBalanceID,
		destination: c.revenueBalanceID,
		description: fmt.Sprintf("Platform fee for order %s", req.OrderID),
		meta: map[string]interface{}{
			metaOrderID: req.OrderID.String(),
			metaType:    "platform_fee",
		},
	})
}

// ChargeProcessingFee books Chapa's non-refundable processing charge against
// platform revenue. Overdraft is allowed so a fee larger than the platform's cut
// on a small order shows up as a loss instead of failing.
func (c *Client) ChargeProcessingFee(ctx context.Context, req ProcessingFeeRequest) (string, error) {
	if c.revenueBalanceID == "" || c.processingFeeBalanceID == "" {
		return "", fmt.Errorf("blnk ChargeProcessingFee: revenue and processing fee balances must be configured")
	}
	return c.createTransaction("ChargeProcessingFee", txnSpec{
		reference:      ProcessingFeeReference(req.OrderID),
		amountCents:    req.AmountCents,
		source:         c.revenueBalanceID,
		destination:    c.processingFeeBalanceID,
		description:    fmt.Sprintf("Chapa processing fee for order %s", req.OrderID),
		allowOverdraft: true,
		meta: map[string]interface{}{
			metaOrderID: req.OrderID.String(),
			metaType:    "processing_fee",
		},
	})
}

// DebitMerchantForReversal claws back a settled sale after the buyer's payment was
// reversed. The merchant balance may go negative; future sales repay it.
func (c *Client) DebitMerchantForReversal(ctx context.Context, req ReversalRequest) (string, error) {
	return c.createTransaction("DebitMerchantForReversal", txnSpec{
		reference:      ReversalReference(req.OrderID),
		amountCents:    req.AmountCents,
		source:         req.MerchantBlnkID,
		destination:    worldSource,
		description:    fmt.Sprintf("Payment reversal for order %s", req.OrderID),
		allowOverdraft: true,
		meta: map[string]interface{}{
			metaOrderID:    req.OrderID.String(),
			metaMerchantID: req.MerchantID.String(),
			metaType:       "reversal",
		},
	})
}

// HoldPayout reserves the payout amount as an inflight debit so it stops counting
// toward the merchant's available balance until the transfer settles.
func (c *Client) HoldPayout(ctx context.Context, req PayoutRequest) (string, error) {
	return c.createTransaction("HoldPayout", txnSpec{
		reference:   PayoutReference(req.PayoutID),
		amountCents: req.AmountCents,
		source:      req.MerchantBlnkID,
		destination: worldSource,
		description: fmt.Sprintf("Payout to %s", req.Destination),
		inflight:    true,
		meta: map[string]interface{}{
			"payout_id":    req.PayoutID.String(),
			metaMerchantID: req.MerchantID.String(),
			"method":       req.Method,
			"destination":  req.Destination,
			metaType:       "payout",
		},
	})
}

func (c *Client) CommitInflight(ctx context.Context, inflightTxnID string) error {
	_, _, err := c.sdk.Transaction.Update(inflightTxnID, blnkgo.UpdateStatus{Status: blnkgo.InflightStatusCommit, SkipQueue: true})
	if err != nil {
		if hasErrorCode(err, blnkgo.ErrorCodeTxnAlreadyCommitted) {
			slog.Info("blnk: inflight already committed", "txn_id", inflightTxnID)
			return nil
		}
		if hasErrorCode(err, blnkgo.ErrorCodeGenConflict) {
			return fmt.Errorf("blnk CommitInflight %s: %w", inflightTxnID, ErrInflightBusy)
		}
		slog.Error("blnk: CommitInflight failed", "txn_id", inflightTxnID, "error", err)
		return fmt.Errorf("blnk CommitInflight: %w", err)
	}
	slog.Info("blnk: inflight committed", "txn_id", inflightTxnID)
	return nil
}

func (c *Client) VoidInflight(ctx context.Context, inflightTxnID string) error {
	_, _, err := c.sdk.Transaction.Update(inflightTxnID, blnkgo.UpdateStatus{Status: blnkgo.InflightStatusVoid, SkipQueue: true})
	if err != nil {
		if hasErrorCode(err, blnkgo.ErrorCodeTxnAlreadyVoided) {
			slog.Info("blnk: inflight already voided", "txn_id", inflightTxnID)
			return nil
		}
		if hasErrorCode(err, blnkgo.ErrorCodeGenConflict) {
			return fmt.Errorf("blnk VoidInflight %s: %w", inflightTxnID, ErrInflightBusy)
		}
		slog.Error("blnk: VoidInflight failed", "txn_id", inflightTxnID, "error", err)
		return fmt.Errorf("blnk VoidInflight: %w", err)
	}
	slog.Info("blnk: inflight voided", "txn_id", inflightTxnID)
	return nil
}

func (c *Client) GetBalance(ctx context.Context, balanceID string) (*Balance, error) {
	balance, _, err := c.sdk.LedgerBalance.Get(balanceID)
	if err != nil {
		slog.Error("blnk: GetBalance failed", "balance_id", balanceID, "error", err)
		return nil, fmt.Errorf("blnk GetBalance: %w", err)
	}

	result := &Balance{
		BalanceCents:        bigToInt64(balance.Balance),
		InflightDebitCents:  bigToInt64(balance.InflightDebitBalance),
		InflightCreditCents: bigToInt64(balance.InflightCreditBalance),
	}
	result.AvailableCents = result.BalanceCents - result.InflightDebitCents
	return result, nil
}

func (c *Client) GetTransactionHistory(ctx context.Context, balanceID string, limit int) ([]TransactionResponse, error) {
	filterResp, _, err := c.sdk.Transaction.Filter(blnkgo.FilterParams{
		Filters: []blnkgo.Filter{
			{Field: "source", Operator: blnkgo.OpEqual, Value: balanceID},
		},
		Limit:     limit,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		slog.Error("blnk: GetTransactionHistory failed", "balance_id", balanceID, "error", err)
		return nil, fmt.Errorf("blnk GetTransactionHistory: %w", err)
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
		result = append(result, TransactionResponse{TransactionID: txnID, Reference: ref, Status: status})
	}
	return result, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c.ledgerID == "" {
		_, _, err := c.sdk.Ledger.List()
		return err
	}
	_, _, err := c.sdk.Ledger.Get(c.ledgerID)
	return err
}

type txnSpec struct {
	reference      string
	amountCents    int64
	source         string
	destination    string
	description    string
	inflight       bool
	allowOverdraft bool
	meta           map[string]interface{}
}

// createTransaction is idempotent on reference: a retry after a lost response
// returns the transaction Blnk already recorded instead of failing.
func (c *Client) createTransaction(op string, spec txnSpec) (string, error) {
	if spec.amountCents <= 0 {
		return "", fmt.Errorf("blnk %s: amount must be positive, got %d", op, spec.amountCents)
	}

	req := blnkgo.CreateTransactionRequest{
		ParentTransaction: blnkgo.ParentTransaction{
			PreciseAmount: big.NewInt(spec.amountCents),
			Precision:     precision,
			Currency:      currencyETB,
			Reference:     spec.reference,
			Source:        spec.source,
			Destination:   spec.destination,
			Description:   spec.description,
			SkipQueue:     true,
			MetaData:      spec.meta,
		},
		Inflight:       spec.inflight,
		AllowOverdraft: spec.allowOverdraft,
	}

	slog.Debug("blnk: creating transaction", "op", op, "reference", spec.reference, "amount_cents", spec.amountCents)

	txn, _, err := c.sdk.Transaction.Create(req)
	if err != nil {
		if hasErrorCode(err, blnkgo.ErrorCodeTxnDuplicateReference) {
			existing, _, getErr := c.sdk.Transaction.GetByReference(spec.reference)
			if getErr != nil {
				return "", fmt.Errorf("blnk %s: duplicate reference %s but lookup failed: %w", op, spec.reference, getErr)
			}
			slog.Info("blnk: transaction already exists for reference", "op", op, "reference", spec.reference, "txn_id", existing.TransactionID)
			return existing.TransactionID, nil
		}
		slog.Error("blnk: transaction failed", "op", op, "reference", spec.reference, "error", err)
		return "", fmt.Errorf("blnk %s: %w", op, err)
	}

	slog.Info("blnk: transaction created", "op", op, "reference", spec.reference, "txn_id", txn.TransactionID)
	return txn.TransactionID, nil
}

func hasErrorCode(err error, code string) bool {
	apiErr, ok := blnkgo.AsApiErrorResponse(err)
	if !ok || apiErr.ErrorDetail == nil {
		return false
	}
	return apiErr.ErrorDetail.Code == code
}

func IsInsufficientFunds(err error) bool {
	return hasErrorCode(err, blnkgo.ErrorCodeTxnInsufficientFunds)
}

func bigToInt64(v *big.Int) int64 {
	if v == nil || !v.IsInt64() {
		return 0
	}
	return v.Int64()
}
