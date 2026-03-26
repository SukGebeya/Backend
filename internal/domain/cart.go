package domain

import (
	"time"

	"github.com/google/uuid"
)

type CartStatus string

const (
	CartStatusActive    CartStatus = "active"
	CartStatusAbandoned CartStatus = "abandoned"
	CartStatusConverted CartStatus = "converted"
)

func (s CartStatus) IsValid() bool {
	return s == CartStatusActive || s == CartStatusAbandoned || s == CartStatusConverted
}

type Cart struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	UserID    *uuid.UUID `db:"user_id" json:"user_id,omitempty"`
	SessionID *string    `db:"session_id" json:"session_id,omitempty"`
	Status    CartStatus `db:"status" json:"status"`
	ExpiresAt time.Time  `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	Items     []CartItem `db:"items" json:"items,omitempty"`
}

type CartItem struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	CartID             uuid.UUID `db:"cart_id" json:"cart_id"`
	VariantID          uuid.UUID `db:"variant_id" json:"variant_id"`
	Quantity           int       `db:"quantity" json:"quantity"`
	PriceSnapshotCents int64     `db:"price_snapshot_cents" json:"price_snapshot_cents"`
	AddedAt            time.Time `db:"added_at" json:"added_at"`
}

type CartWithDetails struct {
	Cart
	Items []CartItemWithDetails `json:"items"`
}

type CartItemWithDetails struct {
	CartItem
	ProductName  string `json:"product_name"`
	VariantTitle string `json:"variant_title"`
	ProductSlug  string `json:"product_slug"`
	MerchantID   string `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	Currency     string `json:"currency"`
}

const MaxCartItems = 50

type AddToCartInput struct {
	UserIDorSession *uuid.UUID
	SessionID       *string
	VariantID       uuid.UUID
	Quantity        int
}

type UpdateCartItemInput struct {
	CartID    uuid.UUID
	VariantID uuid.UUID
	Quantity  int
}

type CartSummary struct {
	ID         uuid.UUID `json:"id"`
	ItemCount  int       `json:"item_count"`
	TotalCents int64     `json:"total_cents"`
	Currency   string    `json:"currency"`
	ExpiresAt  time.Time `json:"expires_at"`
}
