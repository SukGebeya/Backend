package domain

import (
	"time"

	"github.com/google/uuid"
)

type MerchantProfile struct {
	ID              uuid.UUID `db:"id" json:"id"`
	UserID          uuid.UUID `db:"user_id" json:"user_id"`
	ShopName        string    `db:"shop_name" json:"shop_name"`
	ShopSlug        string    `db:"shop_slug" json:"shop_slug"`
	ShopDescription *string   `db:"shop_description" json:"shop_description"`
	ShopLogoURL     *string   `db:"shop_logo_url" json:"shop_logo_url"`
	ShopBannerURL   *string   `db:"shop_banner_url" json:"shop_banner_url"`
	ContactPhone    string    `db:"contact_phone" json:"contact_phone"`
	ContactEmail    *string   `db:"contact_email" json:"contact_email"`
	Address         *string   `db:"address" json:"address"`
	City            string    `db:"city" json:"city"`
	BlnkBalanceID   *string   `db:"blnk_balance_id" json:"blnk_balance_id"`
	BlnkLedgerID    *string   `db:"blnk_ledger_id" json:"blnk_ledger_id"`
	IsVerified      bool      `db:"is_verified" json:"is_verified"`
	TotalSalesCents int64     `db:"total_sales_cents" json:"total_sales_cents"`

	DeliveryFeeCents           int64     `db:"delivery_fee_cents" json:"delivery_fee_cents"`
	FreeDeliveryThresholdCents *int64    `db:"free_delivery_threshold_cents" json:"free_delivery_threshold_cents,omitempty"`
	TIN                        *string   `db:"tin" json:"tin,omitempty"`
	TradeLicenseNumber         *string   `db:"trade_license_number" json:"trade_license_number,omitempty"`
	CreatedAt                  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at" json:"updated_at"`
}

func (m *MerchantProfile) HasWallet() bool {
	return m.BlnkBalanceID != nil && *m.BlnkBalanceID != ""
}

// DeliveryFeeFor returns the merchant's flat delivery fee for an order subtotal,
// or zero when the order qualifies for free delivery.
func (m *MerchantProfile) DeliveryFeeFor(subtotalCents int64) int64 {
	if m.FreeDeliveryThresholdCents != nil && subtotalCents >= *m.FreeDeliveryThresholdCents {
		return 0
	}
	return m.DeliveryFeeCents
}
