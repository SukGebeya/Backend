package domain

import (
	"time"

	"github.com/google/uuid"
)

type WalletTransactionType string

const (
	WalletTransactionTypeSaleCredit    WalletTransactionType = "sale_credit"
	WalletTransactionTypePayoutDebit   WalletTransactionType = "payout_debit"
	WalletTransactionTypeRefundDebit   WalletTransactionType = "refund_debit"
	WalletTransactionTypeReversalDebit WalletTransactionType = "reversal_debit"
)

type WalletTransactionStatus string

const (
	WalletTransactionStatusInflight WalletTransactionStatus = "inflight"
	WalletTransactionStatusApplied  WalletTransactionStatus = "applied"
	WalletTransactionStatusRejected WalletTransactionStatus = "rejected"
)

type WalletTransaction struct {
	ID                uuid.UUID               `db:"id" json:"id"`
	MerchantID        uuid.UUID               `db:"merchant_id" json:"merchant_id"`
	OrderID           *uuid.UUID              `db:"order_id" json:"order_id,omitempty"`
	Type              WalletTransactionType   `db:"type" json:"type"`
	AmountCents       int64                   `db:"amount_cents" json:"amount_cents"`
	Currency          string                  `db:"currency" json:"currency"`
	BlnkTransactionID string                  `db:"blnk_transaction_id" json:"blnk_transaction_id"`
	Status            WalletTransactionStatus `db:"status" json:"status"`
	Description       *string                 `db:"description" json:"description,omitempty"`
	CreatedAt         time.Time               `db:"created_at" json:"created_at"`
}

type CreateWalletTransactionInput struct {
	MerchantID        uuid.UUID
	OrderID           *uuid.UUID
	Type              WalletTransactionType
	AmountCents       int64
	Currency          string
	BlnkTransactionID string
	Status            WalletTransactionStatus
	Description       *string
}

type WalletBalance struct {
	BalanceCents   int64  `json:"balance_cents"`
	InflightCents  int64  `json:"inflight_debit_cents"`
	AvailableCents int64  `json:"available_cents"`
	Currency       string `json:"currency"`
}

type PayoutStatus string

const (
	PayoutStatusPending    PayoutStatus = "pending"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted  PayoutStatus = "completed"
	PayoutStatusFailed     PayoutStatus = "failed"
)

type PayoutMethod string

const (
	PayoutMethodTelebirr     PayoutMethod = "telebirr"
	PayoutMethodBankTransfer PayoutMethod = "bank_transfer"
	PayoutMethodCBEBirr      PayoutMethod = "cbe_birr"
)

type Payout struct {
	ID                uuid.UUID    `db:"id" json:"id"`
	MerchantID        uuid.UUID    `db:"merchant_id" json:"merchant_id"`
	AmountCents       int64        `db:"amount_cents" json:"amount_cents"`
	Currency          string       `db:"currency" json:"currency"`
	Method            PayoutMethod `db:"method" json:"method"`
	Destination       string       `db:"destination" json:"destination"`
	Status            PayoutStatus `db:"status" json:"status"`
	AccountName       *string      `db:"account_name" json:"account_name,omitempty"`
	BankCode          *string      `db:"bank_code" json:"bank_code,omitempty"`
	BlnkTransactionID *string      `db:"blnk_transaction_id" json:"blnk_transaction_id,omitempty"`
	TransferReference *string      `db:"transfer_reference" json:"transfer_reference,omitempty"`
	FailureReason     *string      `db:"failure_reason" json:"failure_reason,omitempty"`
	RequestedAt       time.Time    `db:"requested_at" json:"requested_at"`
	ProcessedAt       *time.Time   `db:"processed_at" json:"processed_at,omitempty"`
}

type CreatePayoutInput struct {
	MerchantID  uuid.UUID
	AmountCents int64
	Currency    string
	Method      PayoutMethod
	Destination string
}

type RequestPayoutInput struct {
	AmountCents int64        `json:"amount_cents" validate:"required,min=100"`
	Method      PayoutMethod `json:"method" validate:"required,oneof=telebirr bank_transfer cbe_birr"`
	Destination string       `json:"destination" validate:"required"`
	AccountName string       `json:"account_name" validate:"required,min=2,max=100"`
	BankCode    string       `json:"bank_code" validate:"required"`
}

type ProcessPayoutInput struct {
	PayoutID uuid.UUID `json:"payout_id"`
}
