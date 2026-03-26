# Gebeya Marketplace Platform - Implementation Plan

>Generated from GEBEYA_PROJECT_PLAN.md analysis
> Document version: 1.0

---

## Executive Summary

Transform the existing Go boilerplate into a production-ready SME marketplace platform for Ethiopian merchants with escrow-backed payments via Blnk and Chapa integration.

### Current State Analysis

**Existing Boilerplate:**
- Chi router with JWT auth (RSA RS256)
- PostgreSQL + pgx/v5 +Redis
- SQLC workflow + Goose migrations
- Basic User (email/phone + role) and Note domain models
- Role system: "user"/"admin"

**To Remove:**
- `Note` domain (not relevant to this product)

**To Extend:**
- User model→ add KYC status, full_name, avatar_url, national_id, is_active
- Role system → add "buyer", "merchant"
- 15+ new domain models with 14 new migrations

---

## Phase 1A — Foundation (Week 1)

**Goal:** Auth + merchant onboarding with Blnk wallet creation

**Status:** ✅ COMPLETE

### Tasks

- [x] Remove `Note` domain files
  - Delete `internal/domain/note.go`
  - Delete `internal/service/note_service.go`
  - Delete `internal/http/handler/note_handler.go`
  - Delete `internal/repository/note_repository.go`
  - Remove Note routes from `router.go`
  - Remove Note queries from `sqlc/queries/queries.sql`
  
- [x] Create migration `00002_extend_users.sql`
  - Add columns: full_name, avatar_url, kyc_status, national_id, is_active
  - Update `internal/domain/user.go` with new fields
  - Update Role constants: add "buyer", "merchant"

- [x] Create migration `00003_create_merchant_profiles.sql`
  - merchant_profiles table with Blnk wallet references

- [x] Create domain model `internal/domain/merchant.go`
  - MerchantProfile struct with all fields from schema

- [x] Update SQLC queries
  - Create `sqlc/queries/users.sql` (move user queries)
  - Create `sqlc/queries/merchants.sql`

- [x] Create Blnk client
  - `internal/blnk/client.go`
  - Methods: CreateMerchantWallet, CreateEscrowHold, CommitEscrow, CreditMerchant, VoidEscrow

- [x] Create Repository layer
  - `internal/repository/merchant_repository.go`
  - Update `internal/repository/interfaces.go`

- [x] Create Service layer
  - `internal/service/merchant_service.go`
  - Update `internal/service/interfaces.go`

- [x] Create Handler layer
  - `internal/http/handler/merchant_handler.go`
  - Add routes to `router.go`

- [x] Update AuthService
  - Accept role parameter ("buyer" or "merchant") in Register

- [ ] Write integration tests for merchant onboarding

### Files Created/Modified

```
migrations/
  00002_extend_users.sql
  00003_create_merchant_profiles.sql

internal/domain/
  user.go (modify)
  merchant.go (new)

internal/blnk/
  client.go (new)

internal/repository/
  interfaces.go (modify)
  merchant_repository.go (new)

internal/service/
  interfaces.go (modify)
  merchant_service.go (new)
  auth_service.go (modify)

internal/http/handler/
  merchant_handler.go (new)

internal/http/
  router.go (modify)

sqlc/queries/
  users.sql
  merchants.sql
```

---

## Phase 1B — Product Catalog (Week 1-2)

**Goal:** Merchants can list products; buyers can browse

**Status:** ✅ COMPLETE

### Tasks

- [x] Run migrations
  - `00004_create_categories.sql`
  - `00005_create_products.sql`
  - `00006_create_product_variants.sql`
  - `00007_create_inventory.sql`

- [x] Create domain models
  - `internal/domain/category.go`
  - `internal/domain/product.go`
  - `internal/domain/inventory.go`

- [x] Create SQLC queries
  - `sqlc/queries/categories.sql`
  - `sqlc/queries/products.sql`
  - `sqlc/queries/inventory.sql`

- [x] Create Repository layer
  - `internal/repository/category_repository.go`
  - `internal/repository/product_repository.go`
  - `internal/repository/inventory_repository.go`

- [x] Create Service layer
  - `internal/service/category_service.go`
  - `internal/service/product_service.go`
  - `internal/service/inventory_service.go`

- [x] Create Handler layer
  - `internal/http/handler/category_handler.go`
  - `internal/http/handler/product_handler.go`

- [x] Implement inventory reservation logic
  - On cart add: reserve quantity
  - On cart remove: release reservation
  - On order ship: decrement quantity and reservation

- [x] Add routes to router.go
  - Public: GET /products, GET /products/{slug}, GET /categories, GET /shops/{shopSlug}
  - Merchant: CRUD /merchant/products

- [ ] Write unit tests for inventory reservation

### Files Created

```
migrations/
  00004_create_categories.sql
  00005_create_products.sql
  00006_create_product_variants.sql
  00007_create_inventory.sql

internal/domain/
  category.go
  product.go
  inventory.go

internal/repository/
  category_repository.go
  product_repository.go
  inventory_repository.go

internal/service/
  category_service.go
  product_service.go
  inventory_service.go

internal/http/handler/
  category_handler.go
  product_handler.go

sqlc/queries/
  categories.sql
  products.sql
  inventory.sql
```

---

## Phase 1C — Cart (Week 2)

**Goal:** Buyers can add items to cart with inventory reservation

**Status:** ✅ COMPLETE

### Tasks

- [x] Run migrations
  - `00008_create_addresses.sql` (note: actually `00009` in implementation)
  - `00009_create_carts.sql` (note: actually `00010` in implementation)
  - `00010_create_cart_items.sql` (note: actually `00011` in implementation)

- [x] Create domain models
  - `internal/domain/address.go`
  - `internal/domain/cart.go`

- [x] Create SQLC queries
  - `sqlc/queries/addresses.sql`
  - `sqlc/queries/carts.sql`

- [x] Create Repository layer
  - `internal/repository/address_repository.go`
  - `internal/repository/cart_repository.go`

- [x] Create Service layer
  - `internal/service/cart_service.go`

- [x] Implement cart merge logic
  - Guest cart merges into user cart on login
  - Take higher quantity for duplicate variants

- [x] Create Handler layer
  - `internal/http/handler/cart_handler.go`

- [x] Add routes
  - GET /cart, POST /cart/items, PUT /cart/items/{variantId}, DELETE /cart/items/{variantId}

- [x] Implement River job: `ReleaseCartReservationWorker`
  - Scheduled for cart expiry (24 hours)

- [ ] Write tests for cart merge and reservation

### Files Created

```
migrations/
  00009_create_addresses.sql
  00010_create_carts.sql
  00011_create_cart_items.sql

internal/domain/
  address.go
  cart.go

internal/repository/
  address_repository.go
  cart_repository.go

internal/service/
  cart_service.go

internal/http/handler/
  cart_handler.go

internal/jobs/
  release_cart_reservation.go

sqlc/queries/
  addresses.sql
  carts.sql
```

---

## Phase 1D — Checkout & Payments (Week 2-3)

**Goal:** End-to-end payment flow working

**Status:** ✅ COMPLETE

### Tasks

- [x] Run migrations
  - `00012_create_orders.sql`
  - `00013_create_order_items.sql` (note: merged into orders migration)
  - `00014_create_payments.sql` (note: actually `00013` in implementation)

- [x] Create domain models
  - `internal/domain/order.go`
  - `internal/domain/payment.go`

- [x] Create SQLC queries
  - `sqlc/queries/orders.sql`
  - `sqlc/queries/payments.sql`

- [x] Create Chapa client
  - `internal/chapa/client.go`
  - Methods: InitializePayment, VerifySignature

- [x] Extend Blnk client
  - Add escrow operations for checkout

- [x] Create Repository layer
  - `internal/repository/order_repository.go`
  - `internal/repository/payment_repository.go`

- [x] Create Service layer
  - `internal/service/order_service.go`
  - `internal/service/checkout_service.go`

- [x] Create Handler layer
  - `internal/http/handler/checkout_handler.go`
  - `internal/http/handler/webhook_handler.go`

- [x] Add routes
  - POST /orders/checkout
  - POST /webhooks/chapa

- [x] Implement River worker: `ProcessPaymentWebhookWorker`
  - Verify Chapa signature
  - Update order status
  - Commit Blnk escrow

- [ ] Write integration tests with Chapa sandbox

### Files Created

```
migrations/
  00012_create_orders.sql
  00013_create_payments.sql

internal/domain/
  order.go
  payment.go

internal/chapa/
  client.go

internal/blnk/
  client.go (extend)

internal/repository/
  order_repository.go
  payment_repository.go

internal/service/
  order_service.go
  checkout_service.go

internal/http/handler/
  checkout_handler.go
  webhook_handler.go

internal/jobs/
  process_payment_webhook.go
  types.go

sqlc/queries/
  orders.sql
  payments.sql
```

---

## Phase 1E — Order Fulfillment (Week 3)

**Goal:** Orders move through full lifecycle

**Status:** ✅ COMPLETE

### Tasks

- [x] Run migrations
  - `00014_create_shipments.sql`
  - `00015_create_wallet_transactions.sql`

- [x] Create domain models
  - `internal/domain/shipment.go`
  - `internal/domain/wallet.go`

- [x] Create SQLC queries
  - `sqlc/queries/shipments.sql`
  - `sqlc/queries/wallet_transactions.sql`

- [x] Create Repository layer
  - `internal/repository/shipment_repository.go`
  - `internal/repository/wallet_repository.go`

- [x] Extend OrderService
  - Status progression: pending_payment → paid → processing → shipped → delivered
  - Merchant can update status
  - Buyer confirms delivery

- [x] Implement delivery confirmation
  - Commit Blnk escrow
  - Credit merchant wallet (subtotal - platform fee)

- [x] Create Handlers
  - Extend `order_handler.go`

- [x] Add routes
  - Buyer: GET /orders, GET /orders/{id}, POST /orders/{id}/confirm-delivery
  - Merchant: GET /merchant/orders, PUT /merchant/orders/{id}/status

- [x] Implement River worker: `SendOrderNotificationWorker`

- [ ] Write tests

### Files Created

```
migrations/
  00014_create_shipments.sql
  00015_create_wallet_transactions.sql

internal/domain/
  shipment.go
  wallet.go

internal/repository/
  shipment_repository.go
  wallet_repository.go

internal/jobs/
  send_order_notification.go

sqlc/queries/
  shipments.sql
  wallets.sql
```
migrations/
  00002_extend_users.sql
  00003_create_merchant_profiles.sql

internal/domain/
  user.go (modify)
  merchant.go (new)

internal/blnk/
  client.go (new)

internal/repository/
  interfaces.go (modify)
  merchant_repository.go (new)

internal/service/
  interfaces.go (modify)
  merchant_service.go (new)
  auth_service.go (modify)

internal/http/handler/
  merchant_handler.go (new)

internal/http/
  router.go (modify)

sqlc/queries/
  users.sql
  merchants.sql
```

---

## Phase 1B — Product Catalog (Week 1-2)

**Goal:** Merchants can list products; buyers can browse

### Tasks

- [ ] Run migrations
  - `00004_create_categories.sql`
  - `00005_create_products.sql`
  - `00006_create_product_variants.sql`
  - `00007_create_inventory.sql`

- [ ] Create domain models
  - `internal/domain/category.go`
  - `internal/domain/product.go`
  - `internal/domain/inventory.go`

- [ ] Create SQLC queries
  - `sqlc/queries/categories.sql`
  - `sqlc/queries/products.sql`
  - `sqlc/queries/inventory.sql`

- [ ] Create Repository layer
  - `internal/repository/category_repository.go`
  - `internal/repository/product_repository.go`
  - `internal/repository/inventory_repository.go`

- [ ] Create Service layer
  - `internal/service/category_service.go`
  - `internal/service/product_service.go`
  - `internal/service/inventory_service.go`

- [ ] Create Handler layer
  - `internal/http/handler/category_handler.go`
  - `internal/http/handler/product_handler.go`

- [ ] Implement inventory reservation logic
  - On cart add: reserve quantity
  - On cart remove: release reservation
  - On order ship: decrement quantity and reservation

- [ ] Add routes to router.go
  - Public: GET /products, GET /products/{slug}, GET /categories, GET /shops/{shopSlug}
  - Merchant: CRUD /merchant/products

- [ ] Write unit tests for inventory reservation

### Files to Create

```
migrations/
  00004_create_categories.sql
  00005_create_products.sql
  00006_create_product_variants.sql
  00007_create_inventory.sql

internal/domain/
  category.go
  product.go
  inventory.go

internal/repository/
  category_repository.go
  product_repository.go
  inventory_repository.go

internal/service/
  category_service.go
  product_service.go
  inventory_service.go

internal/http/handler/
  category_handler.go
  product_handler.go

sqlc/queries/
  categories.sql
  products.sql
  inventory.sql
```

---

## Phase 1C — Cart (Week 2)

**Goal:** Buyers can add items to cart with inventory reservation

### Tasks

- [ ] Run migrations
  - `00008_create_addresses.sql`
  - `00009_create_carts.sql`
  - `00010_create_cart_items.sql`

- [ ] Create domain models
  - `internal/domain/address.go`
  - `internal/domain/cart.go`

- [ ] Create SQLC queries
  - `sqlc/queries/addresses.sql`
  - `sqlc/queries/carts.sql`

- [ ] Create Repository layer
  - `internal/repository/address_repository.go`
  - `internal/repository/cart_repository.go`

- [ ] Create Service layer
  - `internal/service/cart_service.go`

- [ ] implement cart merge logic
  - Guest cart merges into user cart on login
  - Take higher quantity for duplicate variants

- [ ] Create Handler layer
  - `internal/http/handler/cart_handler.go`

- [ ] Add routes
  - GET /cart, POST /cart/items, PUT /cart/items/{variantId}, DELETE /cart/items/{variantId}

- [ ] Implement River job: `ReleaseCartReservationWorker`
  - Scheduled for cart expiry (24 hours)

- [ ] Write tests for cart merge and reservation

### Files to Create

```
migrations/
  00008_create_addresses.sql
  00009_create_carts.sql
  00010_create_cart_items.sql

internal/domain/
  address.go
  cart.go

internal/repository/
  address_repository.go
  cart_repository.go

internal/service/
  cart_service.go

internal/http/handler/
  cart_handler.go

internal/jobs/
  release_cart_reservation.go

sqlc/queries/
  addresses.sql
  carts.sql
```

---

## Phase 1D — Checkout & Payments (Week 2-3)

**Goal:** End-to-end payment flow working

### Tasks

- [ ] Run migrations
  - `00011_create_orders.sql`
  - `00012_create_order_items.sql`
  - `00013_create_payments.sql`

- [ ] Create domain models
  - `internal/domain/order.go`
  - `internal/domain/payment.go`

- [ ] Create SQLC queries
  - `sqlc/queries/orders.sql`
  - `sqlc/queries/payments.sql`

- [ ] Create Chapa client
  - `internal/chapa/client.go`
  - Methods: InitializePayment, VerifySignature

- [ ] Extend Blnk client
  - Add escrow operations for checkout

- [ ] Create Repository layer
  - `internal/repository/order_repository.go`
  - `internal/repository/payment_repository.go`

- [ ] Create Service layer
  - `internal/service/order_service.go`
  - `internal/service/checkout_service.go`

- [ ] Create Handler layer
  - `internal/http/handler/checkout_handler.go`
  - `internal/http/handler/webhook_handler.go`

- [ ] Add routes
  - POST /orders/checkout
  - POST /webhooks/chapa

- [ ] Implement River worker: `ProcessPaymentWebhookWorker`
  - Verify Chapa signature
  - Update order status
  - Commit Blnk escrow

- [ ] Write integration tests with Chapa sandbox

### Files to Create

```
migrations/
  00011_create_orders.sql
  00012_create_order_items.sql
  00013_create_payments.sql

internal/domain/
  order.go
  payment.go

internal/chapa/
  client.go

internal/blnk/
  client.go (extend)

internal/repository/
  order_repository.go
  payment_repository.go

internal/service/
  order_service.go
  checkout_service.go

internal/http/handler/
  checkout_handler.go
  webhook_handler.go

internal/jobs/
  process_payment_webhook.go
  types.go

sqlc/queries/
  orders.sql
  payments.sql
```

---

## Phase 1E — Order Fulfillment (Week 3)

**Goal:** Orders move through full lifecycle

### Tasks

- [ ] Run migrations
  - `00014_create_shipments.sql`
- `00015_create_wallet_transactions.sql`

- [ ] Create domain models
  - `internal/domain/shipment.go`
  - `internal/domain/wallet.go`

- [ ] Create SQLC queries
  - `sqlc/queries/shipments.sql`
  - `sqlc/queries/wallets.sql`

- [ ] Create Repository layer
  - `internal/repository/shipment_repository.go`
  - `internal/repository/wallet_repository.go`

- [ ] Extend OrderService
  - Status progression: pending_payment → paid → processing → shipped → delivered
  - Merchant can update status
  - Buyer confirms delivery

- [ ] Implement delivery confirmation
  - Commit Blnk escrow
  - Credit merchant wallet (subtotal - platform fee)

- [ ] Create Handlers
  - Extend `order_handler.go`

- [ ] Add routes
  - Buyer: GET /orders, GET /orders/{id}, POST /orders/{id}/confirm-delivery
  - Merchant: GET /merchant/orders, PUT /merchant/orders/{id}/status

- [ ] Implement River worker: `SendOrderNotificationWorker`

- [ ] Write tests

### Files to Create

```
migrations/
  00014_create_shipments.sql
  00015_create_wallet_transactions.sql

internal/domain/
  shipment.go
  wallet.go

internal/repository/
  shipment_repository.go
  wallet_repository.go

internal/jobs/
  send_order_notification.go

sqlc/queries/
  shipments.sql
  wallets.sql
```

---

## Phase 1F — Merchant Wallet & Payouts (Week 3-4)

**Goal:** Merchants can see balance and request withdrawals

**Status:** ✅ COMPLETE

### Tasks

- [x] Run migration
  - `00016_create_payouts.sql`

- [x] Create domain model
  - Update `internal/domain/wallet.go` with Payout

- [x] Create SQLC queries
  - `sqlc/queries/payouts.sql`

- [x] Create Repository layer
  - `internal/repository/payout_repository.go`

- [x] Create Service layer
  - `internal/service/wallet_service.go`

- [x] Create Handler layer
  - `internal/http/handler/wallet_handler.go`
  - `internal/http/handler/admin_handler.go`

- [x] Add routes
  - Merchant: GET /merchant/wallet, GET /merchant/wallet/transactions, POST /merchant/wallet/payout
  - Admin: GET /admin/payouts, POST /admin/payouts/{id}/process

- [x] Implement River worker: `ProcessPayoutWorker`
  - Debit Blnk merchant wallet
  - Call Chapa Transfers API

- [ ] Write tests

### Files Created

```
migrations/
  00016_create_payouts.sql

internal/domain/
  wallet.go (updated with Payout types)

internal/repository/
  payout_repository.go

internal/service/
  wallet_service.go

internal/http/handler/
  wallet_handler.go
  admin_handler.go

internal/jobs/
  process_payout.go (in workers.go)

sqlc/queries/
  payouts.sql
```

---

## Phase 1G — Hardening (Week 4)

**Goal:** Production-ready

**Status:** ⚠️ IN PROGRESS

### Tasks

- [x] Run migration
  - River jobs table already managed by River framework

- [ ] Rate limiting review
  - Tighten limits on checkout and payout endpoints

- [x] Idempotency middleware
  - Create `internal/http/middleware/idempotency.go` ✅
  - Create `internal/repository/idempotency.go` ✅
  - Apply to checkout endpoint (required) ✅
  - Apply to payout endpoint (required) ✅
  - Apply to admin payout processing (optional) ✅
  - Add `Idempotency-Key` header to CORS allowed headers ✅
  - Update Swagger documentation ✅
  - Update test documentation ✅

- [ ] Error handling audit
  - Comprehensive logging for Blnk/Chapa calls
  - Proper error types in `internal/errors`

- [x] Admin KYC endpoints
  - PUT /admin/users/{id}/kyc ✅
  - PUT /admin/merchants/{id}/verify ✅

- [x] Order cancellation
  - POST /orders/{id}/cancel ✅

- [ ] Swagger documentation
  - Add `@swagger` annotations to all new handlers (in progress)

- [ ] Docker Compose
  - Add Blnk service
  - Add Chapa mock for testing

- [ ] Load testing
  - Test checkout flow under load

- [ ] Integration tests
  - Write tests for critical flows

### Files Created/Modified

```
internal/http/middleware/
  idempotency.go (NEW)

internal/repository/
  idempotency.go (NEW)

internal/service/
  user_service.go (updated with UpdateKYCStatus)
  merchant_service.go (updated with Verify)
  order_service.go (updated with CancelOrder)

internal/http/handler/
  admin_handler.go (updated with UpdateKYC, VerifyMerchant)
  order_handler.go (updated with CancelOrder)
```

---

## Additional Features Implemented

### Order Cancellation
- Buyer can cancel orders in pending_payment, paid, or processing status
- Voids Blnk escrow hold if present
- Route: `POST /api/v1/orders/{id}/cancel`

### Admin Features
- Admin can update user KYC status
- Admin can verify/unverify merchants
- Admin can view all users and merchants

---

## Known Issues / Future Work

1. **Inventory Release on Cancellation** - Currently, cancelling an order does not release reserved inventory. This requires coordination with the inventory service.

2. **Refund Flow** - Not implemented. Orders can be cancelled but refunds to buyers are not handled.

3. **Testing** - Integration tests are still limited. More comprehensive testing needed.

4. **Idempotency** - ✅ FIXED. Idempotency middleware is now applied to checkout and payout endpoints:
   - `POST /api/v1/orders/checkout` - Requires Idempotency-Key header
   - `POST /api/v1/merchant/wallet/payout` - Requires Idempotency-Key header
   - `POST /api/v1/admin/payouts/{id}/process` - Optional Idempotency-Key header

5. **Load Testing** - Not performed.

6. **Docker Compose** - Blnk service configured in docker-compose-dev.yml. Chapa mock service not configured.

```
migrations/
  00017_create_river_jobs.sql

internal/http/middleware/
  idempotency.go
```

---

## Technical Decisions Summary

### Financial Rules
1. **Never use float for money** - All amounts as `int64` cents
2. **Database transactions for checkout** - Atomic with River job enqueue
3. **Blnk is source of truth** - Our DB is audit trail only
4. **Log all external API calls** - Debug level for success, Error for failures
5. **Idempotency keys** - Use `order_id` for Chapa/Blnk calls

### Code Conventions
1. **No comments in code** - Unless genuinely non-obvious
2. **Use `internal/errors`** - Never raw errors at HTTP boundary
3. **Use `pkg/response`** - Never `json.Marshal` in handlers
4. **Validate with struct tags** - Fail fast with 400
5. **Compile-time interface verification** - On every implementation

### Database Rules
1. **All queries in sqlc/queries/** - Run `make sqlc-generate`
2. **Never SELECT ***- Always specify columns
3. **Snapshot mutable data** - Product names in order_items
4. **Use TIMESTAMPTZ** - Store in UTC

### API Rules
1. **Webhook always returns 200** - Process async via River
2. **TxRef pattern** - `GEBEYA-{orderID}-{timestamp}`
3. **Cursor pagination** - Not offset

---

## Environment VariablesToAdd

```env
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
```

---

## Dependencies to Add

```go
// go.mod additions
github.com/blnkfinance/blnk-go // Blnk SDK
github.com/riverqueue/river // Background jobs
github.com/riverqueue/riverpgxv5 // River pgx driver
```

---

## Migration Files Summary

| File | Purpose |
|------|---------|
| 00002_extend_users.sql | Add KYC fields to users |
| 00003_create_merchant_profiles.sql | Merchant profiles with Blnk refs |
| 00004_create_categories.sql | Product categories |
| 00005_create_products.sql | Products + product_images |
| 00006_create_product_variants.sql | Product variants (SKUs) |
| 00007_create_inventory.sql | Inventory with reservations |
| 00008_create_addresses.sql | Shipping addresses |
| 00009_create_carts.sql | Shopping carts |
| 00010_create_cart_items.sql | Cart line items |
| 00011_create_orders.sql | Orders with order_status enum |
| 00012_create_order_items.sql | Order line items |
| 00013_create_payments.sql | Payment records |
| 00014_create_shipments.sql | Shipment tracking |
| 00015_create_wallet_transactions.sql | Blnk transaction audit log |
| 00016_create_payouts.sql | Merchant payout requests |
| 00017_create_river_jobs.sql | River job queue table |

---

## Project Structure (Final)

```
cmd/api/main.go

internal/
├── config/
│   └── config.go
├── db/
├── domain/
│   ├── user.go
│   ├── merchant.go
│   ├── category.go
│   ├── product.go
│   ├── inventory.go
│   ├── address.go
│   ├── cart.go
│   ├── order.go
│   ├── payment.go
│   ├── shipment.go
│   └── wallet.go
├── errors/
├── http/
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── merchant_handler.go
│   │   ├── product_handler.go
│   │   ├── category_handler.go
│   │   ├── cart_handler.go
│   │   ├── checkout_handler.go
│   │   ├── order_handler.go
│   │   ├── wallet_handler.go
│   │   ├── webhook_handler.go
│   │   ├── admin_handler.go
│   │   └── health_handler.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── common.go
│   │   ├── rate_limit.go
│   │   ├── timeout.go
│   │   └── idempotency.go
│   └── router.go
├── repository/
│   ├── interfaces.go
│   ├── user_repository.go
│   ├── merchant_repository.go
│   ├── product_repository.go
│   ├── category_repository.go
│   ├── inventory_repository.go
│   ├── address_repository.go
│   ├── cart_repository.go
│   ├── order_repository.go
│   ├── payment_repository.go
│   ├── shipment_repository.go
│   ├── wallet_repository.go
│   ├── payout_repository.go
│   ├── redis.go
│   └── circuit_breaker.go
├── service/
│   ├── interfaces.go
│   ├── auth_service.go
│   ├── user_service.go
│   ├── merchant_service.go
│   ├── product_service.go
│   ├── category_service.go
│   ├── inventory_service.go
│   ├── cart_service.go
│   ├── checkout_service.go
│   ├── order_service.go
│   └── wallet_service.go
├── jobs/
│   ├── types.go
│   ├── process_payment_webhook.go
│   ├── release_cart_reservation.go
│   ├── process_payout.go
│   └── send_order_notification.go
└── testutil/

pkg/
├── jwt/
├── password/
├── logger/
├── response/
└── validation/

blnk/
└── client.go

chapa/
└── client.go

migrations/
└── (all .sqlfiles)

sqlc/queries/
├── users.sql
├── merchants.sql
├── categories.sql
├── products.sql
├── inventory.sql
├── addresses.sql
├── carts.sql
├── orders.sql
├── payments.sql
├── shipments.sql
├── wallets.sql
└── payouts.sql
```

---

## Next Steps

1. **Start with Phase 1A** - Remove Note domain, create user extensions
2. **Follow the migration order strictly** - Each migration depends on previous ones
3. **Implement domain models first** - Then SQLC queries, then repositories, then services
4. **Write tests alongside code** - Don't defer testing
5. **Run `make sqlc-generate` after each SQLC query addition**

---

*This plan is based on GEBEYA_PROJECT_PLAN.md and current boilerplate analysis.*