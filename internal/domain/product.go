package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

func (s ProductStatus) IsValid() bool {
	return s == ProductStatusDraft || s == ProductStatusActive || s == ProductStatusArchived
}

type Product struct {
	ID          uuid.UUID     `db:"id" json:"id"`
	MerchantID  uuid.UUID     `db:"merchant_id" json:"merchant_id"`
	CategoryID  *uuid.UUID    `db:"category_id" json:"category_id,omitempty"`
	Name        string        `db:"name" json:"name"`
	Slug        string        `db:"slug" json:"slug"`
	Description *string       `db:"description" json:"description,omitempty"`
	Status      ProductStatus `db:"status" json:"status"`
	IsDigital   bool          `db:"is_digital" json:"is_digital"`
	CreatedAt   time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at" json:"updated_at"`
}

type ProductImage struct {
	ID        uuid.UUID `db:"id" json:"id"`
	ProductID uuid.UUID `db:"product_id" json:"product_id"`
	URL       string    `db:"url" json:"url"`
	AltText   *string   `db:"alt_text" json:"alt_text,omitempty"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	IsPrimary bool      `db:"is_primary" json:"is_primary"`
}

type ProductVariant struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	ProductID         uuid.UUID      `db:"product_id" json:"product_id"`
	SKU               string         `db:"sku" json:"sku"`
	Title             string         `db:"title" json:"title"`
	PriceCents        int64          `db:"price_cents" json:"price_cents"`
	ComparePriceCents *int64         `db:"compare_price_cents" json:"compare_price_cents,omitempty"`
	Currency          string         `db:"currency" json:"currency"`
	Attributes        map[string]any `db:"attributes" json:"attributes"`
	IsActive          bool           `db:"is_active" json:"is_active"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

func (v *ProductVariant) DisplayPrice() float64 {
	return float64(v.PriceCents) / 100
}

func (v *ProductVariant) HasDiscount() bool {
	return v.ComparePriceCents != nil && *v.ComparePriceCents > v.PriceCents
}

func (v *ProductVariant) DiscountPercent() float64 {
	if !v.HasDiscount() {
		return 0
	}
	return (float64(*v.ComparePriceCents-v.PriceCents) / float64(*v.ComparePriceCents)) * 100
}

type ProductWithVariants struct {
	Product  Product
	Variants []ProductVariant
	Images   []ProductImage
}

type ProductWithInventory struct {
	Product   Product
	Variant   ProductVariant
	Inventory *Inventory
}

type ProductFilter struct {
	CategoryID *uuid.UUID
	MerchantID *uuid.UUID
	Search     string
}

// ProductSummary is a product as shown in listings: its lowest active price, how
// many units can be bought right now, and the shop selling it.
type ProductSummary struct {
	Product
	MinPriceCents     int64   `json:"min_price_cents"`
	AvailableQuantity int64   `json:"available_quantity"`
	PrimaryImageURL   *string `json:"primary_image_url,omitempty"`
	ShopName          string  `json:"shop_name"`
	ShopSlug          string  `json:"shop_slug"`
	MerchantVerified  bool    `json:"merchant_verified"`
}

const MaxImagesPerProduct = 10
