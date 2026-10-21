package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusProcessing     OrderStatus = "processing"
	OrderStatusShipped        OrderStatus = "shipped"
	OrderStatusDelivered      OrderStatus = "delivered"
	OrderStatusCancelled      OrderStatus = "cancelled"
	OrderStatusRefunded       OrderStatus = "refunded"
	OrderStatusDisputed       OrderStatus = "disputed"
)

func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPendingPayment, OrderStatusPaid, OrderStatusProcessing,
		OrderStatusShipped, OrderStatusDelivered, OrderStatusCancelled, OrderStatusRefunded, OrderStatusDisputed:
		return true
	default:
		return false
	}
}

func (s OrderStatus) String() string {
	return string(s)
}

type Order struct {
	ID                uuid.UUID   `db:"id" json:"id"`
	BuyerID           uuid.UUID   `db:"buyer_id" json:"buyer_id"`
	MerchantID        uuid.UUID   `db:"merchant_id" json:"merchant_id"`
	ShippingAddressID *uuid.UUID  `db:"shipping_address_id" json:"shipping_address_id,omitempty"`
	Status            OrderStatus `db:"status" json:"status"`
	SubtotalCents     int64       `db:"subtotal_cents" json:"subtotal_cents"`
	PlatformFeeCents  int64       `db:"platform_fee_cents" json:"platform_fee_cents"`
	TotalCents        int64       `db:"total_cents" json:"total_cents"`
	Currency          string      `db:"currency" json:"currency"`
	Notes             *string     `db:"notes" json:"notes,omitempty"`
	BlnkTransactionID *string     `db:"blnk_transaction_id" json:"blnk_transaction_id,omitempty"`
	DeliveryFeeCents  int64       `db:"delivery_fee_cents" json:"delivery_fee_cents"`
	DisputedAt        *time.Time  `db:"disputed_at" json:"disputed_at,omitempty"`
	DisputeReason     *string     `db:"dispute_reason" json:"dispute_reason,omitempty"`
	ChapaTxRef        *string     `db:"chapa_tx_ref" json:"chapa_tx_ref,omitempty"`
	CreatedAt         time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at" json:"updated_at"`
}

type OrderItem struct {
	ID              uuid.UUID `db:"id" json:"id"`
	OrderID         uuid.UUID `db:"order_id" json:"order_id"`
	VariantID       uuid.UUID `db:"variant_id" json:"variant_id"`
	ProductName     string    `db:"product_name" json:"product_name"`
	VariantTitle    string    `db:"variant_title" json:"variant_title"`
	Quantity        int       `db:"quantity" json:"quantity"`
	UnitPriceCents  int64     `db:"unit_price_cents" json:"unit_price_cents"`
	TotalPriceCents int64     `db:"total_price_cents" json:"total_price_cents"`
}

type OrderWithItems struct {
	Order Order
	Items []OrderItem
}

type CreateOrderInput struct {
	BuyerID           uuid.UUID
	MerchantID        uuid.UUID
	ShippingAddressID *uuid.UUID
	Items             []CreateOrderItemInput
	Notes             *string
}

type CreateOrderItemInput struct {
	VariantID      uuid.UUID
	ProductName    string
	VariantTitle   string
	Quantity       int
	UnitPriceCents int64
}

const DefaultPlatformFeePercent = 5

func CalculatePlatformFee(subtotalCents, feePercent int64) int64 {
	return subtotalCents * feePercent / 100
}

// CalculateMerchantPayout is what the merchant receives at settlement: the goods
// subtotal minus the platform fee, plus the delivery fee they charged in full.
func CalculateMerchantPayout(subtotalCents, platformFeeCents, deliveryFeeCents int64) int64 {
	return subtotalCents - platformFeeCents + deliveryFeeCents
}
