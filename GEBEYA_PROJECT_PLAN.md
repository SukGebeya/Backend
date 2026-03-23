# Gebeya — SME Marketplace Platform
## Agent Context Document & Implementation Plan

> **For AI coding agents:** This document is your primary source of truth for this project. Read it fully before writing any code. Every architectural decision, naming convention, data model, and integration detail is specified here. When in doubt, refer back to this document rather than making assumptions.

---

## Table of Contents

1. [Product Vision](#1-product-vision)
2. [Market Context](#2-market-context)
3. [Boilerplate Foundation](#3-boilerplate-foundation)
4. [System Architecture](#4-system-architecture)
5. [Domain Models & Database Schema](#5-domain-models--database-schema)
6. [API Specification](#6-api-specification)
7. [Blnk Finance Integration](#7-blnk-finance-integration)
8. [Chapa Payment Gateway Integration](#8-chapa-payment-gateway-integration)
9. [Background Jobs with River](#9-background-jobs-with-river)
10. [Implementation Roadmap](#10-implementation-roadmap)
11. [Project Structure](#11-project-structure)
12. [Environment Variables](#12-environment-variables)
13. [Critical Rules & Conventions](#13-critical-rules--conventions)

---

## 1. Product Vision

**Gebeya** (ገበያ — Amharic for "marketplace") is a digital marketplace platform built for small and informal merchants in Addis Ababa, Ethiopia. It gives merchants a simple digital storefront and a Blnk-powered financial wallet, and gives buyers a trusted way to transact with merchants they have never met before using escrow-backed payments.

### The Core Problem

Ethiopia's informal economy is enormous. Thousands of small merchants — tailors, food sellers, artisans, electronics repair shops, home-based businesses — have real customers but no digital presence. They accept cash only, have no transaction history, cannot prove their revenue to access credit, and cannot safely transact with buyers in other cities. The infrastructure (Telebirr, Chapa) now exists to change this. The commerce layer on top of it does not.

### What Gebeya Does

- Gives merchants a digital storefront (shop page, product listings)
- Handles buyer-to-merchant payments via escrow (Chapa for money in, Blnk for holding and releasing)
- Gives merchants a Blnk wallet that accumulates their earnings
- Lets merchants withdraw to their Telebirr or bank account on demand
- Builds a verified transaction history for every merchant — the foundation for working capital products later

### The Three-Phase Business

```
Phase 1 (MVP):   Marketplace + Escrow Wallet
                 Merchant lists products → Buyer pays → Escrow holds → Delivery confirmed → Merchant paid

Phase 2:         Working Capital
                 Merchant's Blnk transaction history unlocks short-term advances repaid from future sales

Phase 3:         Merchant Neobank
                 Full financial product — payroll, supplier payments, tax records, savings pockets
```

This document covers Phase 1 (MVP) in full implementation detail.

---

## 2. Market Context

- Ethiopia averaged over 7.5 million digital transactions daily as of 2024
- Digital transactions exceeded cash transactions for the first time in April 2024
- ~40% of Ethiopian fintech startups are payment processors, but the merchant commerce layer remains underdeveloped
- Telebirr has 40M+ users; Chapa is the leading developer-facing PSP licensed by the National Bank of Ethiopia (NBE)
- The Ethiopian government aims to increase financial inclusion from 46% to 70% of adults

### Competitive Positioning

**Zemen GEBEYA** (Ethio Telecom's marketplace) exists but is a telecom-built product — slow to iterate, enterprise-focused, not built around small merchant UX. Our advantage is execution speed, merchant-first design, and the Blnk wallet layer which Zemen GEBEYA does not have.

### Regulatory Notes

- Payment collection requires partnership with a licensed PSP → use **Chapa** (already NBE-licensed)
- Wallet/balance holding that does not constitute deposit-taking avoids full banking licensing at MVP stage
- Implement KYC (phone + national ID) from day one to stay ahead of NBE requirements
- Do not offer interest-bearing products in Phase 1 (savings interest requires licensing)
- Working capital advances in Phase 2 should be structured as flat-fee products, not interest-bearing loans, until proper licensing is obtained

---

## 3. Boilerplate Foundation

This project starts from a production-ready Go boilerplate. Agents must understand what already exists before adding anything.

### What the Boilerplate Provides (Do Not Rebuild)

| Component | Status | Notes |
|-----------|--------|-------|
| Chi router | ✅ Ready | Use existing router setup in `internal/http/router.go` |
| JWT auth (RSA RS256) | ✅ Ready | Access token (15min) + refresh token (7 days) via Redis |
| PostgreSQL + pgx/v5 | ✅ Ready | Connection pool: max 25, min 5 |
| Redis | ✅ Ready | Used for refresh tokens + rate limiting |
| SQLC workflow | ✅ Ready | Write queries in `sqlc/queries/`, run `make sqlc-generate` |
| Goose migrations | ✅ Ready | Files in `migrations/`, run `make migrate-up` |
| Error types | ✅ Ready | Use `internal/errors` package — never return raw errors to HTTP |
| JSON responses | ✅ Ready | Use `pkg/response` — never write `json.Marshal` in handlers |
| Argon2id password hashing | ✅ Ready | Use `pkg/password` |
| Structured logging | ✅ Ready | Use `pkg/logger` with slog |
| Request validation | ✅ Ready | Use `go-playground/validator` tags on request structs |
| Circuit breaker (Redis) | ✅ Ready | Wraps Redis repos; do not add circuit breaker elsewhere |
| Swagger docs | ✅ Ready | Add `@swagger` annotations to all new handlers |
| Testcontainers setup | ✅ Ready | Use `internal/testutil` for integration tests |
| Uber mock generation | ✅ Ready | Run `make mocks` after adding new interfaces |

### What the Boilerplate Has That We Replace

| Boilerplate Component | Replaced With |
|-----------------------|---------------|
| `Note` domain | Remove entirely — not relevant to this product |
| Basic `User` (email/phone + role) | Extend with merchant/buyer profile fields |
| Simple role ("user"/"admin") | Extend to "buyer", "merchant", "admin" |

### Boilerplate Conventions to Follow Strictly

- **Handler pattern**: Parse → Validate → Call service → Write response. No business logic in handlers.
- **Service pattern**: Business logic only. Calls repository and external services (Blnk, Chapa). Returns domain types.
- **Repository pattern**: Data access only. Uses SQLC-generated code. Maps DB rows to domain types.
- **Error handling**: Always use `internal/errors` types. Never return `fmt.Errorf` or raw `error` strings to HTTP layer.
- **No comments in code** unless explicitly complex logic requires it.
- **Interface verification** at compile time: `var _ ServiceInterface = (*ServiceImpl)(nil)`

---

## 4. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client (Web / Mobile)                         │
└─────────────────────────────┬───────────────────────────────────┘
                              │ HTTPS
┌─────────────────────────────▼───────────────────────────────────┐
│              Gebeya Go API (Chi Router)                          │
│                                                                  │
│  Middleware chain:                                               │
│  RequestID → RealIP → Timeout(30s) → Logger →                   │
│  Recoverer → CORS → RateLimit → Auth (protected routes)         │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Handler   │  │   Handler   │  │   Handler   │            │
│  │  (auth)     │  │ (products)  │  │  (orders)   │  ...       │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
│         └────────────────┼────────────────┘                     │
│                          │                                       │
│  ┌───────────────────────▼──────────────────────────────────┐   │
│  │                    Service Layer                          │   │
│  │  AuthService / MerchantService / ProductService /        │   │
│  │  OrderService / WalletService / CheckoutService          │   │
│  └──────────┬───────────────────────────┬───────────────────┘   │
│             │                           │                        │
│  ┌──────────▼──────────┐   ┌────────────▼──────────────────┐   │
│  │   Repository Layer  │   │   External Service Clients    │   │
│  │  (SQLC + pgx)       │   │   BlnkClient / ChapaClient    │   │
│  └──────────┬──────────┘   └────────────┬──────────────────┘   │
└─────────────┼────────────────────────────┼─────────────────────┘
              │                            │
   ┌──────────▼──────────┐     ┌──────────▼──────────────────┐
   │   PostgreSQL 16      │     │   Blnk Finance (self-hosted) │
   │   + Redis 7          │     │   + Chapa PSP (API)          │
   └─────────────────────┘     └─────────────────────────────┘
                                            │
                               ┌────────────▼──────────────────┐
                               │   River (background jobs)      │
                               │   (Postgres-backed)            │
                               └───────────────────────────────┘
```

### Service Responsibilities

| Service | Responsibility |
|---------|---------------|
| `AuthService` | Register, login, token refresh, logout. Extends boilerplate. |
| `MerchantService` | Merchant profile management, KYC status, Blnk wallet creation |
| `ProductService` | Product + variant CRUD, image management, category management |
| `InventoryService` | Stock levels, reservation logic (quantity − reserved = available) |
| `CartService` | Add/remove items, guest + authenticated carts, cart merge on login |
| `OrderService` | Cart → order conversion, order status lifecycle |
| `CheckoutService` | Orchestrates: validate cart → create order → initiate Chapa payment → create Blnk inflight transaction |
| `WalletService` | Merchant wallet balance queries, payout requests, transaction history |
| `WebhookService` | Handles Chapa payment webhooks → confirms Blnk inflight transaction |

---

## 5. Domain Models & Database Schema

### Migration Order

Migrations must be created in this exact sequence (each file is a goose `.sql` file):

```
00001_extend_users.sql
00002_create_merchant_profiles.sql
00003_create_categories.sql
00004_create_products.sql
00005_create_product_variants.sql
00006_create_inventory.sql
00007_create_carts.sql
00008_create_cart_items.sql
00009_create_orders.sql
00010_create_order_items.sql
00011_create_payments.sql
00012_create_shipments.sql
00013_create_wallet_transactions.sql
00014_create_payouts.sql
00015_create_river_jobs.sql
```

---

### 5.1 Extend Users Table

The boilerplate `users` table has: `id`, `email`, `phone`, `password_hash`, `role`, `created_at`, `updated_at`.

```sql
-- +goose Up
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS full_name      TEXT,
  ADD COLUMN IF NOT EXISTS avatar_url     TEXT,
  ADD COLUMN IF NOT EXISTS kyc_status     TEXT NOT NULL DEFAULT 'pending',
  ADD COLUMN IF NOT EXISTS national_id    TEXT,
  ADD COLUMN IF NOT EXISTS is_active      BOOLEAN NOT NULL DEFAULT true;

-- kyc_status values: 'pending', 'submitted', 'verified', 'rejected'
-- role values (already exists): 'buyer', 'merchant', 'admin'

-- +goose Down
ALTER TABLE users
  DROP COLUMN IF EXISTS full_name,
  DROP COLUMN IF EXISTS avatar_url,
  DROP COLUMN IF EXISTS kyc_status,
  DROP COLUMN IF EXISTS national_id,
  DROP COLUMN IF EXISTS is_active;
```

---

### 5.2 Merchant Profiles

One-to-one with users where role = 'merchant'. Contains all merchant-specific data including the Blnk wallet reference.

```sql
-- +goose Up
CREATE TABLE merchant_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    shop_name           TEXT NOT NULL,
    shop_slug           TEXT NOT NULL UNIQUE,
    shop_description    TEXT,
    shop_logo_url       TEXT,
    shop_banner_url     TEXT,
    contact_phone       TEXT NOT NULL,
    contact_email       TEXT,
    address             TEXT,
    city                TEXT NOT NULL DEFAULT 'Addis Ababa',
    blnk_balance_id     TEXT UNIQUE,        -- Blnk balance ID for merchant wallet
    blnk_ledger_id      TEXT,               -- Blnk ledger this wallet belongs to
    is_verified         BOOLEAN NOT NULL DEFAULT false,
    total_sales_cents   BIGINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_merchant_profiles_user_id ON merchant_profiles(user_id);
CREATE INDEX idx_merchant_profiles_shop_slug ON merchant_profiles(shop_slug);

-- +goose Down
DROP TABLE IF EXISTS merchant_profiles;
```

**Domain model:**
```go
// internal/domain/merchant.go
type MerchantProfile struct {
    ID               uuid.UUID  `db:"id"`
    UserID           uuid.UUID  `db:"user_id"`
    ShopName         string     `db:"shop_name"`
    ShopSlug         string     `db:"shop_slug"`
    ShopDescription  *string    `db:"shop_description"`
    ShopLogoURL      *string    `db:"shop_logo_url"`
    ContactPhone     string     `db:"contact_phone"`
    ContactEmail     *string    `db:"contact_email"`
    Address          *string    `db:"address"`
    City             string     `db:"city"`
    BlnkBalanceID    *string    `db:"blnk_balance_id"`
    BlnkLedgerID     *string    `db:"blnk_ledger_id"`
    IsVerified       bool       `db:"is_verified"`
    TotalSalesCents  int64      `db:"total_sales_cents"`
    CreatedAt        time.Time  `db:"created_at"`
    UpdatedAt        time.Time  `db:"updated_at"`
}
```

---

### 5.3 Categories

```sql
-- +goose Up
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id   UUID REFERENCES categories(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    icon_url    TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO categories (name, slug, sort_order) VALUES
  ('Clothing & Fashion', 'clothing-fashion', 1),
  ('Food & Beverages', 'food-beverages', 2),
  ('Electronics', 'electronics', 3),
  ('Crafts & Handmade', 'crafts-handmade', 4),
  ('Beauty & Personal Care', 'beauty-personal-care', 5),
  ('Home & Living', 'home-living', 6),
  ('Services', 'services', 7),
  ('Other', 'other', 8);

-- +goose Down
DROP TABLE IF EXISTS categories;
```

---

### 5.4 Products

```sql
-- +goose Up
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id     UUID NOT NULL REFERENCES merchant_profiles(id) ON DELETE CASCADE,
    category_id     UUID REFERENCES categories(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'draft',
    is_digital      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(merchant_id, slug)
);

-- status values: 'draft', 'active', 'archived'

CREATE TABLE product_images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    alt_text    TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    is_primary  BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_products_merchant_id ON products(merchant_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_status ON products(status);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);

-- +goose Down
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS products;
```

---

### 5.5 Product Variants

Every purchasable item is a variant. Even a product with no options (e.g. a single injera price) must have at least one variant.

```sql
-- +goose Up
CREATE TABLE product_variants (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id              UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku                     TEXT NOT NULL UNIQUE,
    title                   TEXT NOT NULL DEFAULT 'Default',
    price_cents             BIGINT NOT NULL,
    compare_price_cents     BIGINT,
    currency                TEXT NOT NULL DEFAULT 'ETB',
    attributes              JSONB NOT NULL DEFAULT '{}',
    is_active               BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- attributes JSONB stores variant options e.g.:
-- {"size": "XL", "color": "White"} or {"weight": "500g"}

CREATE INDEX idx_product_variants_product_id ON product_variants(product_id);

-- +goose Down
DROP TABLE IF EXISTS product_variants;
```

**CRITICAL MONEY RULE — enforce in all agents:**
> Never use `FLOAT` or `DECIMAL` for money. Always use `BIGINT` storing the amount in the smallest currency unit (cents for ETB, i.e. santim). 1 ETB = 100 santim. Store 250 ETB as `25000`. Display by dividing by 100.

**Domain model:**
```go
// internal/domain/product.go
type ProductVariant struct {
    ID                 uuid.UUID      `db:"id"`
    ProductID          uuid.UUID      `db:"product_id"`
    SKU                string         `db:"sku"`
    Title              string         `db:"title"`
    PriceCents         int64          `db:"price_cents"`
    ComparePriceCents  *int64         `db:"compare_price_cents"`
    Currency           string         `db:"currency"`
    Attributes         map[string]any `db:"attributes"`
    IsActive           bool           `db:"is_active"`
    CreatedAt          time.Time      `db:"created_at"`
    UpdatedAt          time.Time      `db:"updated_at"`
}

// DisplayPrice returns ETB amount as float for display only — never use for calculations
func (v *ProductVariant) DisplayPrice() float64 {
    return float64(v.PriceCents) / 100
}
```

---

### 5.6 Inventory

```sql
-- +goose Up
CREATE TABLE inventory (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id  UUID NOT NULL UNIQUE REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity    INT NOT NULL DEFAULT 0,
    reserved    INT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_quantity_positive CHECK (quantity >= 0),
    CONSTRAINT chk_reserved_positive CHECK (reserved >= 0),
    CONSTRAINT chk_reserved_lte_quantity CHECK (reserved <= quantity)
);

-- +goose Down
DROP TABLE IF EXISTS inventory;
```

**Inventory reservation logic (enforce in InventoryService):**
```go
// Available = quantity - reserved
// On cart add:      UPDATE inventory SET reserved = reserved + $qty WHERE variant_id = $id AND (quantity - reserved) >= $qty
// On order confirm: no change (still reserved)
// On order cancel:  UPDATE inventory SET reserved = reserved - $qty
// On order ship:    UPDATE inventory SET quantity = quantity - $qty, reserved = reserved - $qty
```

---

### 5.7 Addresses

```sql
-- +goose Up
CREATE TABLE addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name       TEXT NOT NULL,
    phone           TEXT NOT NULL,
    line1           TEXT NOT NULL,
    line2           TEXT,
    city            TEXT NOT NULL DEFAULT 'Addis Ababa',
    subcity         TEXT,
    woreda          TEXT,
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);

-- +goose Down
DROP TABLE IF EXISTS addresses;
```

> Note: Ethiopian addresses use subcity and woreda subdivisions, not postal codes. Design address fields accordingly.

---

### 5.8 Carts

```sql
-- +goose Up
CREATE TABLE carts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id  TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cart_items (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id                 UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    variant_id              UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity                INT NOT NULL DEFAULT 1,
    price_snapshot_cents    BIGINT NOT NULL,
    added_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(cart_id, variant_id)
);

CREATE INDEX idx_carts_user_id ON carts(user_id);
CREATE INDEX idx_carts_session_id ON carts(session_id);
CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);

-- +goose Down
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS carts;
```

**Cart merge logic (implement in CartService):**
When a guest user logs in, merge their session cart into their user cart:
1. Find session cart by `session_id`
2. Find user cart by `user_id`
3. For each item in session cart: if variant exists in user cart, take the higher quantity; else insert the item
4. Delete session cart after merge

---

### 5.9 Orders

```sql
-- +goose Up
CREATE TYPE order_status AS ENUM (
    'pending_payment',
    'paid',
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'refunded'
);

CREATE TABLE orders (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id                UUID NOT NULL REFERENCES users(id),
    merchant_id             UUID NOT NULL REFERENCES merchant_profiles(id),
    shipping_address_id     UUID REFERENCES addresses(id),
    status                  order_status NOT NULL DEFAULT 'pending_payment',
    subtotal_cents          BIGINT NOT NULL,
    platform_fee_cents      BIGINT NOT NULL,
    total_cents             BIGINT NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'ETB',
    notes                   TEXT,
    blnk_transaction_id     TEXT,           -- Blnk inflight transaction ID
    chapa_tx_ref            TEXT UNIQUE,    -- Chapa payment reference
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id          UUID NOT NULL REFERENCES product_variants(id),
    product_name        TEXT NOT NULL,
    variant_title       TEXT NOT NULL,
    quantity            INT NOT NULL,
    unit_price_cents    BIGINT NOT NULL,
    total_price_cents   BIGINT NOT NULL
);

-- Snapshot product name + variant title at order time so historical orders
-- remain accurate even if the merchant later renames/deletes the product.

CREATE INDEX idx_orders_buyer_id ON orders(buyer_id);
CREATE INDEX idx_orders_merchant_id ON orders(merchant_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_chapa_tx_ref ON orders(chapa_tx_ref);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- +goose Down
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;
```

**Platform fee calculation:**
```go
const PlatformFeePercent = 5 // 5% of subtotal

func CalculateFee(subtotalCents int64) int64 {
    return subtotalCents * PlatformFeePercent / 100
}
// Merchant receives: subtotal - platform_fee
// total_cents = subtotal_cents (buyer pays subtotal; fee is taken from merchant payout)
```

---

### 5.10 Payments

```sql
-- +goose Up
CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id),
    provider            TEXT NOT NULL DEFAULT 'chapa',
    status              TEXT NOT NULL DEFAULT 'pending',
    amount_cents        BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    provider_tx_ref     TEXT UNIQUE,
    provider_response   JSONB,
    blnk_balance_id     TEXT,   -- Blnk escrow balance that received funds
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- status values: 'pending', 'succeeded', 'failed', 'refunded'

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_provider_tx_ref ON payments(provider_tx_ref);

-- +goose Down
DROP TABLE IF EXISTS payments;
```

---

### 5.11 Shipments

```sql
-- +goose Up
CREATE TABLE shipments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id),
    status              TEXT NOT NULL DEFAULT 'preparing',
    carrier             TEXT,
    tracking_number     TEXT,
    estimated_delivery  DATE,
    shipped_at          TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- status values: 'preparing', 'shipped', 'out_for_delivery', 'delivered', 'failed'

CREATE INDEX idx_shipments_order_id ON shipments(order_id);

-- +goose Down
DROP TABLE IF EXISTS shipments;
```

---

### 5.12 Wallet Transactions

Records every Blnk-backed financial event for auditability inside our own DB, even though Blnk is the source of truth for balances.

```sql
-- +goose Up
CREATE TABLE wallet_transactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id         UUID NOT NULL REFERENCES merchant_profiles(id),
    order_id            UUID REFERENCES orders(id),
    type                TEXT NOT NULL,
    amount_cents        BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    blnk_transaction_id TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'applied',
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- type values: 'sale_credit', 'payout_debit', 'refund_debit', 'advance_credit', 'advance_repayment_debit'
-- status values: 'inflight', 'applied', 'rejected'

CREATE INDEX idx_wallet_transactions_merchant_id ON wallet_transactions(merchant_id);
CREATE INDEX idx_wallet_transactions_order_id ON wallet_transactions(order_id);

-- +goose Down
DROP TABLE IF EXISTS wallet_transactions;
```

---

### 5.13 Payouts

```sql
-- +goose Up
CREATE TABLE payouts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id         UUID NOT NULL REFERENCES merchant_profiles(id),
    amount_cents        BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    method              TEXT NOT NULL,
    destination         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    blnk_transaction_id TEXT,
    failure_reason      TEXT,
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at        TIMESTAMPTZ
);

-- method values: 'telebirr', 'bank_transfer', 'cbe_birr'
-- status values: 'pending', 'processing', 'completed', 'failed'

CREATE INDEX idx_payouts_merchant_id ON payouts(merchant_id);
CREATE INDEX idx_payouts_status ON payouts(status);

-- +goose Down
DROP TABLE IF EXISTS payouts;
```

---

## 6. API Specification

### Route Groups

```go
// internal/http/router.go additions

r.Route("/api/v1", func(r chi.Router) {
    // Public
    r.Post("/register", authHandler.Register)
    r.Post("/login", authHandler.Login)
    r.Post("/refresh", authHandler.Refresh)
    r.Post("/logout", authHandler.Logout)

    r.Get("/products", productHandler.ListProducts)
    r.Get("/products/{slug}", productHandler.GetProduct)
    r.Get("/shops/{shopSlug}", merchantHandler.GetShop)
    r.Get("/categories", categoryHandler.ListCategories)

    r.Post("/webhooks/chapa", webhookHandler.HandleChapa) // no auth — verified by signature

    // Authenticated (any role)
    r.Group(func(r chi.Router) {
        r.Use(AuthMiddleware)

        r.Get("/users/me", userHandler.GetMe)
        r.Put("/users/me", userHandler.UpdateMe)

        r.Get("/cart", cartHandler.GetCart)
        r.Post("/cart/items", cartHandler.AddItem)
        r.Put("/cart/items/{variantId}", cartHandler.UpdateItem)
        r.Delete("/cart/items/{variantId}", cartHandler.RemoveItem)

        r.Post("/orders/checkout", checkoutHandler.Checkout)
        r.Get("/orders", orderHandler.ListMyOrders)
        r.Get("/orders/{id}", orderHandler.GetOrder)
        r.Post("/orders/{id}/confirm-delivery", orderHandler.ConfirmDelivery)
    })

    // Merchant routes
    r.Group(func(r chi.Router) {
        r.Use(AuthMiddleware)
        r.Use(RoleMiddleware("merchant"))

        r.Post("/merchant/onboard", merchantHandler.Onboard)
        r.Put("/merchant/profile", merchantHandler.UpdateProfile)
        r.Get("/merchant/dashboard", merchantHandler.GetDashboard)

        r.Get("/merchant/products", productHandler.ListMyProducts)
        r.Post("/merchant/products", productHandler.CreateProduct)
        r.Put("/merchant/products/{id}", productHandler.UpdateProduct)
        r.Delete("/merchant/products/{id}", productHandler.DeleteProduct)
        r.Post("/merchant/products/{id}/variants", productHandler.AddVariant)

        r.Get("/merchant/orders", orderHandler.ListMerchantOrders)
        r.Put("/merchant/orders/{id}/status", orderHandler.UpdateOrderStatus)

        r.Get("/merchant/wallet", walletHandler.GetWallet)
        r.Get("/merchant/wallet/transactions", walletHandler.ListTransactions)
        r.Post("/merchant/wallet/payout", walletHandler.RequestPayout)
    })

    // Admin routes
    r.Group(func(r chi.Router) {
        r.Use(AuthMiddleware)
        r.Use(RoleMiddleware("admin"))

        r.Get("/admin/users", adminHandler.ListUsers)
        r.Put("/admin/users/{id}/kyc", adminHandler.UpdateKYC)
        r.Get("/admin/merchants", adminHandler.ListMerchants)
        r.Put("/admin/merchants/{id}/verify", adminHandler.VerifyMerchant)
        r.Get("/admin/orders", adminHandler.ListAllOrders)
        r.Get("/admin/payouts", adminHandler.ListPayouts)
        r.Post("/admin/payouts/{id}/process", adminHandler.ProcessPayout)
    })
})
```

### Key Request/Response Shapes

**POST /api/v1/merchant/onboard**
```json
Request:
{
  "shop_name": "Tigist Habesha Designs",
  "shop_slug": "tigist-habesha",
  "contact_phone": "+251911234567",
  "city": "Addis Ababa",
  "shop_description": "Traditional Ethiopian clothing handmade in Addis"
}

Response 201:
{
  "merchant_id": "uuid",
  "shop_slug": "tigist-habesha",
  "blnk_balance_id": "bln_xxxxx",
  "message": "Merchant profile created. KYC verification pending."
}
```

**POST /api/v1/orders/checkout**
```json
Request:
{
  "shipping_address_id": "uuid",
  "notes": "Please wrap carefully"
}

Response 201:
{
  "order_id": "uuid",
  "total_cents": 265000,
  "currency": "ETB",
  "payment_url": "https://checkout.chapa.co/checkout/payment/xxxxx",
  "chapa_tx_ref": "GEBEYA-uuid-timestamp"
}
```

**POST /api/v1/webhooks/chapa** (Chapa calls this)
```json
{
  "event": "charge.success",
  "data": {
    "tx_ref": "GEBEYA-uuid-timestamp",
    "amount": "2650.00",
    "currency": "ETB",
    "status": "success"
  }
}
```

---

## 7. Blnk Finance Integration

Blnk is a self-hosted, double-entry ledger system. It is the financial source of truth for all money movement in Gebeya. The Go API communicates with Blnk via its REST API using the official Go SDK (`blnkfinance/blnk-go`).

### Blnk Concepts

| Blnk Concept | Gebeya Usage |
|---|---|
| **Ledger** | One ledger per currency (ETB ledger). Created once at system setup. |
| **Balance** | One balance per merchant = their wallet. One escrow balance for the platform. |
| **Transaction** | Every money movement: buyer payment into escrow, escrow release to merchant, payout debit |
| **Inflight Transaction** | Funds held in escrow — committed when delivery confirmed, voided on cancellation |

### Blnk Client Setup

```go
// internal/blnk/client.go
package blnk

import (
    blnkgo "github.com/blnkfinance/blnk-go"
)

type Client struct {
    sdk *blnkgo.BlnkClient
}

func NewClient(baseURL, secretKey string) *Client {
    sdk := blnkgo.NewBlnkClient(baseURL, secretKey)
    return &Client{sdk: sdk}
}
```

### Ledger Structure

```
Gebeya ETB Ledger
├── Platform Escrow Balance        (one per platform — holds buyer payments)
├── Platform Revenue Balance       (accumulates platform fees)
├── Merchant Balance: Tigist       (blnk_balance_id stored in merchant_profiles)
├── Merchant Balance: Bekele       (blnk_balance_id stored in merchant_profiles)
└── ... one per merchant
```

### Blnk Operations by Event

**1. Merchant Onboarding — Create Wallet**
```go
// Called from MerchantService.Onboard()
func (c *Client) CreateMerchantWallet(ctx context.Context, merchantID, shopName string) (balanceID string, err error) {
    balance, _, err := c.sdk.Balance.Create(ctx, blnkgo.CreateBalanceRequest{
        LedgerID:   os.Getenv("BLNK_ETB_LEDGER_ID"),
        Currency:   "ETB",
        MetaData: map[string]any{
            "merchant_id": merchantID,
            "shop_name":   shopName,
            "type":        "merchant_wallet",
        },
    })
    return balance.BalanceID, err
}
```

**2. Checkout — Create Inflight Transaction (Escrow)**
```go
// Called from CheckoutService.Checkout() AFTER creating the order record
// This moves buyer funds from "incoming" into platform escrow, held inflight
func (c *Client) CreateEscrowHold(ctx context.Context, req EscrowHoldRequest) (txnID string, err error) {
    txn, _, err := c.sdk.Transaction.Create(ctx, blnkgo.CreateTransactionRequest{
        Amount:          float64(req.AmountCents) / 100,
        Currency:        "ETB",
        Precision:       100,
        Reference:       req.OrderID.String(),
        Source:          "@WorldUSD",              // Blnk internal source for incoming funds
        Destination:     os.Getenv("BLNK_ESCROW_BALANCE_ID"),
        Description:     fmt.Sprintf("Escrow for order %s", req.OrderID),
        AllowOverdraft:  true,
        Inflight:        true,                     // CRITICAL: holds funds, not yet committed
        MetaData: map[string]any{
            "order_id":    req.OrderID,
            "merchant_id": req.MerchantID,
        },
    })
    return txn.TransactionID, err
}
```

**3. Delivery Confirmed — Commit Inflight + Credit Merchant**

This is a two-step operation triggered when the buyer confirms delivery:

```go
// Step 1: Commit the inflight escrow transaction
func (c *Client) CommitEscrow(ctx context.Context, inflightTxnID string) error {
    _, _, err := c.sdk.Transaction.UpdateInflight(ctx, inflightTxnID, blnkgo.UpdateInflightRequest{
        Status: "commit",
    })
    return err
}

// Step 2: Move net amount (after fee) from escrow to merchant wallet
func (c *Client) CreditMerchant(ctx context.Context, req MerchantCreditRequest) (txnID string, err error) {
    netAmountCents := req.SubtotalCents - req.FeeCents
    txn, _, err := c.sdk.Transaction.Create(ctx, blnkgo.CreateTransactionRequest{
        Amount:      float64(netAmountCents) / 100,
        Currency:    "ETB",
        Precision:   100,
        Reference:   fmt.Sprintf("payout-%s", req.OrderID),
        Source:      os.Getenv("BLNK_ESCROW_BALANCE_ID"),
        Destination: req.MerchantBlnkBalanceID,
        Description: fmt.Sprintf("Sale credit for order %s", req.OrderID),
        MetaData: map[string]any{
            "order_id":    req.OrderID,
            "merchant_id": req.MerchantID,
        },
    })
    return txn.TransactionID, err
}
```

**4. Order Cancelled — Void Inflight**
```go
func (c *Client) VoidEscrow(ctx context.Context, inflightTxnID string) error {
    _, _, err := c.sdk.Transaction.UpdateInflight(ctx, inflightTxnID, blnkgo.UpdateInflightRequest{
        Status: "void",
    })
    return err
}
```

**5. Payout — Debit Merchant Wallet**
```go
func (c *Client) DebitMerchantForPayout(ctx context.Context, req PayoutRequest) (txnID string, err error) {
    txn, _, err := c.sdk.Transaction.Create(ctx, blnkgo.CreateTransactionRequest{
        Amount:      float64(req.AmountCents) / 100,
        Currency:    "ETB",
        Precision:   100,
        Reference:   req.PayoutID.String(),
        Source:      req.MerchantBlnkBalanceID,
        Destination: "@WorldUSD",   // Blnk internal sink for outgoing funds
        Description: fmt.Sprintf("Payout to %s", req.Destination),
        MetaData: map[string]any{
            "payout_id":   req.PayoutID,
            "merchant_id": req.MerchantID,
            "method":      req.Method,
            "destination": req.Destination,
        },
    })
    return txn.TransactionID, err
}
```

### Blnk Environment Variables

```env
BLNK_BASE_URL=http://localhost:5001
BLNK_SECRET_KEY=your-blnk-secret-key
BLNK_ETB_LEDGER_ID=ldg_xxxxxxxx          # Created once at system setup
BLNK_ESCROW_BALANCE_ID=bln_xxxxxxxx      # Platform escrow wallet
BLNK_REVENUE_BALANCE_ID=bln_xxxxxxxx     # Platform revenue wallet
```

---

## 8. Chapa Payment Gateway Integration

Chapa is Ethiopia's leading PSP, licensed by NBE. It handles the actual fiat money movement (Telebirr, bank transfer, card).

**Base URL:** `https://api.chapa.co/v1`

### Chapa Client

```go
// internal/chapa/client.go
package chapa

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type Client struct {
    secretKey  string
    baseURL    string
    httpClient *http.Client
}

func NewClient(secretKey string) *Client {
    return &Client{
        secretKey:  secretKey,
        baseURL:    "https://api.chapa.co/v1",
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

### Initialize Payment

Called during checkout. Returns a `payment_url` to redirect the buyer to.

```go
type InitializeRequest struct {
    Amount      string `json:"amount"`       // ETB amount as string e.g. "2650.00"
    Currency    string `json:"currency"`     // "ETB"
    Email       string `json:"email"`
    FirstName   string `json:"first_name"`
    LastName    string `json:"last_name"`
    Phone       string `json:"phone_number"`
    TxRef       string `json:"tx_ref"`       // "GEBEYA-{orderID}-{timestamp}"
    CallbackURL string `json:"callback_url"` // Your webhook endpoint
    ReturnURL   string `json:"return_url"`   // Frontend redirect after payment
    Description string `json:"customization[description]"`
}

type InitializeResponse struct {
    Message string `json:"message"`
    Status  string `json:"status"`
    Data    struct {
        CheckoutURL string `json:"checkout_url"`
    } `json:"data"`
}
```

### Webhook Verification

Chapa sends a webhook to `/api/v1/webhooks/chapa` after successful payment. Verify the signature before processing:

```go
// internal/http/handler/webhook_handler.go
func (h *WebhookHandler) HandleChapa(w http.ResponseWriter, r *http.Request) {
    // 1. Verify Chapa-Signature header
    signature := r.Header.Get("Chapa-Signature")
    if !h.verifyChapaSignature(signature, body) {
        response.Unauthorized(w, "invalid webhook signature")
        return
    }

    // 2. Parse event
    // 3. Find order by tx_ref
    // 4. Enqueue River job for async processing (do not process inline)
    _, err := h.riverClient.Insert(r.Context(), ProcessPaymentWebhookArgs{
        TxRef:  event.TxRef,
        Status: event.Status,
    }, nil)

    // 5. Always return 200 immediately to Chapa
    response.OK(w, map[string]string{"received": "true"})
}
```

**CRITICAL:** Always return HTTP 200 to Chapa immediately. Process the webhook asynchronously via River. Chapa will retry if it receives a non-200 response, causing duplicate processing.

---

## 9. Background Jobs with River

River is a Postgres-backed job queue. It runs inside the same Go process and uses the same PostgreSQL database. Install: `github.com/riverqueue/river`.

### Job Types

```go
// internal/jobs/types.go

// ProcessPaymentWebhookArgs: triggered by Chapa webhook
// → verify payment → commit Blnk escrow → update order status
type ProcessPaymentWebhookArgs struct {
    TxRef  string `json:"tx_ref"`
    Status string `json:"status"`
}

// ReleaseCartReservationArgs: triggered when cart expires
// → decrement inventory.reserved for all items in cart
type ReleaseCartReservationArgs struct {
    CartID uuid.UUID `json:"cart_id"`
}

// ProcessPayoutArgs: triggered by merchant payout request
// → call Blnk to debit merchant wallet → call Chapa to send to Telebirr/bank
type ProcessPayoutArgs struct {
    PayoutID uuid.UUID `json:"payout_id"`
}

// SendOrderNotificationArgs: triggered on order status changes
// → send SMS via Africa's Talking or similar
type SendOrderNotificationArgs struct {
    OrderID uuid.UUID `json:"order_id"`
    Event   string    `json:"event"` // "payment_confirmed", "shipped", "delivered"
    Phone   string    `json:"phone"`
}
```

### Transactional Job Enqueueing (Critical Pattern)

Always enqueue River jobs inside the same database transaction as the business operation:

```go
// internal/service/checkout_service.go
func (s *CheckoutService) Checkout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return nil, errors.Internal("failed to begin transaction", err)
    }
    defer tx.Rollback(ctx)

    // 1. Create order record (inside tx)
    order, err := s.queries.WithTx(tx).CreateOrder(ctx, ...)

    // 2. Initialize Chapa payment (outside tx — external call)
    chapaResp, err := s.chapaClient.InitializePayment(ctx, ...)

    // 3. Create Blnk inflight transaction (outside tx — external call)
    blnkTxnID, err := s.blnkClient.CreateEscrowHold(ctx, ...)

    // 4. Update order with external references (inside tx)
    err = s.queries.WithTx(tx).UpdateOrderPaymentRefs(ctx, ...)

    // 5. Enqueue cart expiry job atomically with the order creation
    _, err = s.riverClient.InsertTx(ctx, tx, ReleaseCartReservationArgs{CartID: req.CartID}, &river.InsertOpts{
        ScheduledAt: time.Now().Add(24 * time.Hour), // cancel reservation if not paid in 24h
    })

    tx.Commit(ctx)
    return &CheckoutResult{PaymentURL: chapaResp.CheckoutURL}, nil
}
```

### River Setup in main.go

```go
// Add to cmd/api/main.go after DB initialization

riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
    Queues: map[string]river.QueueConfig{
        river.QueueDefault: {MaxWorkers: 10},
    },
    Workers: river.NewWorkers(),
})
river.AddWorker(riverClient.Workers, &ProcessPaymentWebhookWorker{...})
river.AddWorker(riverClient.Workers, &ReleaseCartReservationWorker{...})
river.AddWorker(riverClient.Workers, &ProcessPayoutWorker{...})
river.AddWorker(riverClient.Workers, &SendOrderNotificationWorker{...})

if err := riverClient.Start(ctx); err != nil {
    logger.Fatal("failed to start river", "error", err)
}
defer riverClient.Stop(ctx)
```

---

## 10. Implementation Roadmap

Build in this exact order. Each phase is a working, testable state.

### Phase 1A — Foundation (Week 1)
Goal: Working auth + merchant onboarding with Blnk wallet creation

- [ ] Remove `Note` domain from boilerplate (delete files, remove routes)
- [ ] Run migration `00001_extend_users.sql`
- [ ] Run migration `00002_create_merchant_profiles.sql`
- [ ] Add SQLC queries for user extensions and merchant profiles
- [ ] Implement `MerchantService.Onboard()` — creates DB record + Blnk wallet
- [ ] Add `POST /api/v1/merchant/onboard` endpoint
- [ ] Update `AuthService.Register()` to accept role: "buyer" or "merchant"
- [ ] Write integration tests for merchant onboarding flow

### Phase 1B — Product Catalog (Week 1-2)
Goal: Merchants can list products; buyers can browse

- [ ] Run migrations `00003` through `00006` (categories, products, variants, inventory)
- [ ] SQLC queries: product CRUD, variant CRUD, inventory management
- [ ] Implement `ProductService` — create/update/delete products and variants
- [ ] Implement `InventoryService` — stock management with reservation logic
- [ ] Public endpoints: list products, get product, list categories, get shop
- [ ] Merchant endpoints: manage own products and variants
- [ ] Write unit tests for inventory reservation logic

### Phase 1C — Cart (Week 2)
Goal: Buyers can add items to cart

- [ ] Run migrations `00007_create_carts.sql`, `00008_create_cart_items.sql`
- [ ] SQLC queries for cart operations
- [ ] Implement `CartService` — add/remove/update items, guest cart, merge on login
- [ ] Cart endpoints (authenticated + guest via session)
- [ ] Inventory reservation on cart add, release on cart remove
- [ ] River job: `ReleaseCartReservationArgs` for expired carts
- [ ] Write tests for cart merge and inventory reservation

### Phase 1D — Checkout & Payments (Week 2-3)
Goal: End-to-end payment flow working

- [ ] Run migrations `00009` through `00011` (orders, order items, payments)
- [ ] Implement Chapa client (`internal/chapa/client.go`)
- [ ] Implement Blnk client (`internal/blnk/client.go`)
- [ ] Implement `CheckoutService` — full transactional checkout
- [ ] `POST /api/v1/orders/checkout` endpoint
- [ ] `POST /api/v1/webhooks/chapa` endpoint + webhook signature verification
- [ ] River worker: `ProcessPaymentWebhookWorker`
- [ ] Order status progression: pending_payment → paid
- [ ] Write integration tests using Chapa sandbox credentials

### Phase 1E — Order Fulfillment (Week 3)
Goal: Orders move through the full lifecycle

- [ ] Run migrations `00012_create_shipments.sql`, `00013_create_wallet_transactions.sql`
- [ ] Merchant: view orders, update status to "processing" and "shipped"
- [ ] Buyer: confirm delivery → triggers Blnk escrow commit + merchant wallet credit
- [ ] River worker: `SendOrderNotificationWorker`
- [ ] `GET /api/v1/orders` and `GET /api/v1/merchant/orders` endpoints

### Phase 1F — Merchant Wallet & Payouts (Week 3-4)
Goal: Merchants can see their balance and request withdrawals

- [ ] Run migration `00014_create_payouts.sql`
- [ ] `GET /api/v1/merchant/wallet` — queries Blnk for live balance + our DB for history
- [ ] `POST /api/v1/merchant/wallet/payout` — creates payout record, enqueues River job
- [ ] River worker: `ProcessPayoutWorker` — debits Blnk, calls Chapa Transfers API
- [ ] Admin: view and manually process payout requests
- [ ] Write tests for payout flow

### Phase 1G — Hardening (Week 4)
Goal: Production-ready

- [ ] Rate limiting review — tighten limits on checkout and payout endpoints
- [ ] Idempotency keys on all Chapa and Blnk calls
- [ ] Comprehensive error handling and logging for all financial operations
- [ ] Admin KYC endpoints
- [ ] Full Swagger documentation
- [ ] Docker Compose production config with Blnk service included
- [ ] Load test checkout flow

---

## 11. Project Structure

The complete structure extending the boilerplate:

```
cmd/api/main.go

internal/
├── config/
│   └── config.go              # Add BLNK_*, CHAPA_* env vars
│
├── db/                        # GENERATED by sqlc — never edit
│
├── domain/
│   ├── user.go                # Extended user + KYC status
│   ├── merchant.go            # MerchantProfile
│   ├── product.go             # Product, ProductVariant, ProductImage
│   ├── inventory.go           # Inventory
│   ├── cart.go                # Cart, CartItem
│   ├── order.go               # Order, OrderItem, OrderStatus
│   ├── payment.go             # Payment
│   ├── shipment.go            # Shipment
│   └── wallet.go              # WalletTransaction, Payout
│
├── errors/                    # Unchanged from boilerplate
│
├── http/
│   ├── handler/
│   │   ├── auth_handler.go    # Extended from boilerplate
│   │   ├── user_handler.go    # Extended from boilerplate
│   │   ├── merchant_handler.go
│   │   ├── product_handler.go
│   │   ├── cart_handler.go
│   │   ├── checkout_handler.go
│   │   ├── order_handler.go
│   │   ├── wallet_handler.go
│   │   ├── webhook_handler.go
│   │   ├── admin_handler.go
│   │   └── health_handler.go  # Extend to check Blnk health
│   ├── middleware/
│   │   ├── auth.go            # Unchanged from boilerplate
│   │   ├── role.go            # RoleMiddleware — already exists, ensure used
│   │   └── idempotency.go     # New: idempotency key middleware for payments
│   └── router.go              # Extended with all new routes
│
├── repository/
│   ├── interfaces.go          # Add new repository interfaces
│   ├── user_repository.go     # Extended
│   ├── merchant_repository.go
│   ├── product_repository.go
│   ├── inventory_repository.go
│   ├── cart_repository.go
│   ├── order_repository.go
│   ├── payment_repository.go
│   ├── wallet_repository.go
│   ├── redis.go               # Unchanged
│   ├── circuit_breaker.go     # Unchanged
│   └── mocks/                 # GENERATED
│
├── service/
│   ├── interfaces.go          # Add new service interfaces
│   ├── auth_service.go        # Extended from boilerplate
│   ├── user_service.go        # Extended from boilerplate
│   ├── merchant_service.go
│   ├── product_service.go
│   ├── inventory_service.go
│   ├── cart_service.go
│   ├── checkout_service.go    # Orchestrates Chapa + Blnk + DB
│   ├── order_service.go
│   ├── wallet_service.go
│   └── mocks/                 # GENERATED
│
├── jobs/
│   ├── process_payment_webhook.go
│   ├── release_cart_reservation.go
│   ├── process_payout.go
│   ├── send_order_notification.go
│   └── types.go
│
└── testutil/                  # Unchanged from boilerplate

pkg/
├── jwt/                       # Unchanged
├── password/                  # Unchanged
├── logger/                    # Unchanged
├── response/                  # Unchanged
└── validation/                # Add phone validator for Ethiopian +251 numbers

blnk/
└── client.go                  # Blnk API client wrapper

chapa/
└── client.go                  # Chapa API client wrapper

migrations/
│   (all .sql files as listed in section 5)

sqlc/queries/
│   ├── users.sql
│   ├── merchants.sql
│   ├── products.sql
│   ├── inventory.sql
│   ├── carts.sql
│   ├── orders.sql
│   ├── payments.sql
│   └── wallets.sql

docs/                          # Generated Swagger
```

---

## 12. Environment Variables

Full `.env` extending the boilerplate `.env.example`:

```env
# Server (from boilerplate)
PORT=8080
ENVIRONMENT=development

# Database (from boilerplate)
DATABASE_URL=postgres://postgres@localhost:5432/gebeya?sslmode=disable

# Redis (from boilerplate)
REDIS_URL=redis://localhost:6379

# JWT (from boilerplate)
JWT_ACCESS_PRIVATE_KEY_PATH=keys/access_private.pem
JWT_ACCESS_PUBLIC_KEY_PATH=keys/access_public.pem
JWT_REFRESH_PRIVATE_KEY_PATH=keys/refresh_private.pem
JWT_REFRESH_PUBLIC_KEY_PATH=keys/refresh_public.pem
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Blnk Finance (NEW)
BLNK_BASE_URL=http://localhost:5001
BLNK_SECRET_KEY=your-blnk-secret-key
BLNK_ETB_LEDGER_ID=ldg_xxxxxxxxxxxxxxxx
BLNK_ESCROW_BALANCE_ID=bln_xxxxxxxxxxxxxxxx
BLNK_REVENUE_BALANCE_ID=bln_xxxxxxxxxxxxxxxx

# Chapa (NEW)
CHAPA_SECRET_KEY=CHASECK_TEST-xxxxxxxxxxxxxxxx
CHAPA_WEBHOOK_SECRET=your-chapa-webhook-secret
CHAPA_CALLBACK_URL=https://yourdomain.com/api/v1/webhooks/chapa
CHAPA_RETURN_URL=https://yourdomain.com/payment/complete

# Platform Config (NEW)
PLATFORM_FEE_PERCENT=5
PLATFORM_NAME=Gebeya

# Rate Limiting (from boilerplate)
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Logging (from boilerplate)
LOG_FORMAT=json
LOG_LEVEL=info
```

---

## 13. Critical Rules & Conventions

These rules apply to every agent working on this codebase. They are non-negotiable.

### Financial Rules
1. **Never use float for money.** All amounts are stored and calculated as `int64` cents. Divide by 100 only for display or when calling external APIs that require decimal strings.
2. **Always use database transactions for checkout.** The order creation, inventory update, and River job enqueue must be atomic.
3. **Blnk is the financial source of truth.** Our `wallet_transactions` table is an audit log, not the balance. Always query Blnk for live balance figures.
4. **Log every Blnk and Chapa API call** with request/response details at debug level, and failures at error level with full context.
5. **Idempotency keys on payment operations.** Use `order_id` as the idempotency key for Chapa and Blnk calls. Never initiate a duplicate charge.

### Code Rules (from boilerplate)
6. **No comments in code** unless the logic is genuinely non-obvious (complex financial calculations, concurrency patterns).
7. **Use `internal/errors` for all errors returned to HTTP handlers.** Never `fmt.Errorf` or `errors.New` at the handler boundary.
8. **Use `pkg/response` for all JSON responses.** Never call `json.Marshal` or `w.Write` directly in handlers.
9. **Validate all incoming requests** with `go-playground/validator` struct tags. Fail fast with 400 before hitting the service layer.
10. **Compile-time interface verification** on every service and repository implementation.

### Database Rules
11. **Write all queries in `sqlc/queries/*.sql`**, run `make sqlc-generate`, never hand-write DB interaction code.
12. **Never SELECT *.** Always specify columns in SQLC queries.
13. **Snapshot mutable data in orders.** `order_items` must store `product_name` and `variant_title` as TEXT columns, copied at order creation time. Product names change; order history must not.
14. **Use `TIMESTAMPTZ`** (not `TIMESTAMP`) for all time columns. Ethiopia is UTC+3; store in UTC always.

### API Rules
15. **Webhook endpoint must always return HTTP 200 to Chapa**, even on internal errors. Log failures and enqueue a retry job instead.
16. **Use `chapa_tx_ref` pattern:** `GEBEYA-{orderID}-{unixTimestamp}` — globally unique, traceable back to the order.
17. **Paginate all list endpoints.** Use `limit` and `cursor` (UUID-based cursor pagination, not offset — more performant at scale).

### Testing Rules
18. **Use `internal/testutil` testcontainers** for integration tests — never mock the database layer.
19. **Mock external services** (Blnk client, Chapa client) in service unit tests using the generated Uber mocks.
20. **Test the inventory reservation logic thoroughly** — it's the most critical non-financial business logic and the most likely source of overselling bugs.

---

*Document version: 1.0 — covers Phase 1 (MVP) in full. Phase 2 (working capital) and Phase 3 (neobank) to be documented after MVP launch.*
