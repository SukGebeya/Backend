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
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

func (m *MerchantProfile) HasWallet() bool {
	return m.BlnkBalanceID != nil && *m.BlnkBalanceID != ""
}
