package domain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusSucceeded, PaymentStatusFailed, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

func (s PaymentStatus) String() string {
	return string(s)
}

type Payment struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	OrderID           uuid.UUID      `db:"order_id" json:"order_id"`
	Provider          string         `db:"provider" json:"provider"`
	Status            PaymentStatus  `db:"status" json:"status"`
	AmountCents       int64          `db:"amount_cents" json:"amount_cents"`
	Currency          string         `db:"currency" json:"currency"`
	ProviderTxRef     *string        `db:"provider_tx_ref" json:"provider_tx_ref,omitempty"`
	ProviderResponse  map[string]any `db:"provider_response" json:"provider_response,omitempty"`
	BlnkTransactionID *string        `db:"blnk_transaction_id" json:"blnk_transaction_id,omitempty"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

type CreatePaymentInput struct {
	OrderID     uuid.UUID
	Provider    string
	AmountCents int64
	Currency    string
}
