# SukGebeya API Testing Guide

## Prerequisites

### 1. Start Docker Services

```bash
docker compose -f docker-compose-dev.yml up -d
```

Wait for all services to be healthy:

```bash
docker compose -f docker-compose-dev.yml ps
```

### 2. Run Database Migrations

```bash
make migrate-up
```

### 3. Seed Development Data

```bash
make seed-dev
```

### 4. Generate SQLC Code (if needed)

```bash
make sqlc-generate
```

### 5. Start the API Server

```bash
make run
# or with hot reload:
make air
```

### 6. Verify Server is Running

```bash
curl http://localhost:8080/health
# Expected: {"status":"healthy"}
```

---

## Test Accounts

| Role | Email | Password | User ID |
|------|-------|----------|---------|
| Admin | admin@test.com | Admin123! | `00000000-0000-0000-0000-000000000001` |
| Buyer | buyer@test.com | Password123! | `00000000-0000-0000-0000-000000000002` |
| Merchant | merchant@test.com | Password123! | `00000000-0000-0000-0000-000000000003` |

**Merchant Profile ID**: `20000000-0000-0000-0000-000000000001`

---

## Automated Testing

### Run Full Test Suite

```bash
chmod +x scripts/test-api.sh
./scripts/test-api.sh all
```

### Run Specific Phase

```bash
./scripts/test-api.sh 1-auth      # Authentication
./scripts/test-api.sh 2-public    # Public endpoints
./scripts/test-api.sh 3-cart      # Cart operations
./scripts/test-api.sh 4-merchant  # Merchant operations
./scripts/test-api.sh 5-admin     # Admin operations
./scripts/test-api.sh 6-checkout  # Checkout flow
./scripts/test-api.sh 7-wallet    # Wallet & payouts
```

---

## Manual Testing

### Phase 1: Authentication

#### 1a. Login as Admin

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"admin@sukgebeya.com","password":"Admin123!"}'
```

**Expected Response:**
```json
{
  "user": {
    "id": "00000000-0000-0000-0000-000000000001",
    "email": "admin@sukgebeya.com",
    "role": "admin",
    ...
  },
  "token": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 900
  }
}
```

Save the `access_token` for subsequent requests.

#### 1b. Login as Buyer

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"buyer@test.com","password":"Password123!"}'
```

#### 1c. Login as Merchant

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"identifier":"merchant@test.com","password":"Password123!"}'
```

#### 1d. Register New User

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email":"newuser@example.com","password":"Password123!","role":"buyer"}'
```

#### 1e. Refresh Token

```bash
curl -X POST http://localhost:8080/api/v1/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<your-refresh-token>"}'
```

#### 1f. Logout

```bash
curl -X POST http://localhost:8080/api/v1/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<your-refresh-token>"}'
```

---

### Phase 2: Public Endpoints

#### 2a. Health Check

```bash
curl http://localhost:8080/health
```

#### 2b. List Categories

```bash
curl http://localhost:8080/api/v1/categories
```

#### 2c. Get Category by ID

```bash
curl http://localhost:8080/api/v1/categories/10000000-0000-0000-0000-000000000001
```

#### 2d. Get Category by Slug

```bash
curl http://localhost:8080/api/v1/categories/slug/clothing-fashion
```

#### 2e. List Products

```bash
curl http://localhost:8080/api/v1/products
```

#### 2f. Get Product by Slug

```bash
curl http://localhost:8080/api/v1/products/test-product-1
```

#### 2g. Get Shop by Slug

```bash
curl http://localhost:8080/api/v1/shops/test-fashion-shop
```

---

### Phase 3: Cart Operations (Guest Session)

#### 3a. Get Cart (Creates Guest Session)

```bash
curl -v http://localhost:8080/api/v1/cart 2>&1 | grep -i "x-session-id"
```

The response header `X-Session-ID` is your guest session. Save it for subsequent requests.

#### 3b. Add Item to Cart

```bash
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: <your-session-id>" \
  -d '{"variant_id":"50000000-0000-0000-0000-000000000001","quantity":2}'
```

#### 3c. Get Cart

```bash
curl http://localhost:8080/api/v1/cart \
  -H "X-Session-ID: <your-session-id>"
```

#### 3d. Update Cart Item

```bash
curl -X PUT http://localhost:8080/api/v1/cart/items/50000000-0000-0000-0000-000000000001 \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: <your-session-id>" \
  -d '{"quantity":3}'
```

#### 3e. Remove Cart Item

```bash
curl -X DELETE http://localhost:8080/api/v1/cart/items/50000000-0000-0000-0000-000000000001 \
  -H "X-Session-ID: <your-session-id>"
```

#### 3f. Clear Cart

```bash
curl -X DELETE http://localhost:8080/api/v1/cart \
  -H "X-Session-ID: <your-session-id>"
```

---

### Phase 4: Cart Operations (Authenticated)

#### 4a. Add Item with Authentication

```bash
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -d '{"variant_id":"50000000-0000-0000-0000-000000000001","quantity":1}'
```

#### 4b. Get Authenticated Cart

```bash
curl http://localhost:8080/api/v1/cart \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 4c. Merge Guest Cart (after login)

```bash
curl -X POST http://localhost:8080/api/v1/cart/merge \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -H "X-Session-ID: <guest-session-id>"
```

---

### Phase 5: Address Management

#### 5a. Create Address

```bash
curl -X POST http://localhost:8080/api/v1/addresses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -d '{
    "full_name": "John Doe",
    "phone": "+251911234567",
    "line1": "Bole Road",
    "line2": "Apt 123",
    "city": "Addis Ababa",
    "subcity": "Bole",
    "woreda": "03",
    "is_default": true
  }'
```

#### 5b. List Addresses

```bash
curl http://localhost:8080/api/v1/addresses \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 5c. Get Address by ID

```bash
curl http://localhost:8080/api/v1/addresses/<address-id> \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 5d. Update Address

```bash
curl -X PUT http://localhost:8080/api/v1/addresses/<address-id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -d '{
    "full_name": "John Doe",
    "phone": "+251911234567",
    "line1": "Mercato Area",
    "line2": "Floor 5",
    "city": "Addis Ababa",
    "subcity": "Arada"
  }'
```

#### 5e. Set Default Address

```bash
curl -X PUT http://localhost:8080/api/v1/addresses/<address-id>/default \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 5f. Delete Address

```bash
curl -X DELETE http://localhost:8080/api/v1/addresses/<address-id> \
  -H "Authorization: Bearer <buyer-access-token>"
```

---

### Phase 6: Merchant Operations

#### 6a. Get Merchant Profile

```bash
curl http://localhost:8080/api/v1/merchant/profile \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 6b. Update Merchant Profile

```bash
curl -X PUT http://localhost:8080/api/v1/merchant/profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -d '{
    "shop_description": "Updated shop description",
    "address": "New Address",
    "city": "Addis Ababa"
  }'
```

#### 6c. List Merchant's Products

```bash
curl http://localhost:8080/api/v1/merchant/products \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 6d. Create Product

```bash
curl -X POST http://localhost:8080/api/v1/merchant/products \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -d '{
    "name": "New Test Product",
    "slug": "new-test-product",
    "description": "A new test product for development",
    "category_id": "10000000-0000-0000-0000-000000000001",
    "is_digital": false
  }'
```

#### 6e. Add Product Variant

```bash
curl -X POST http://localhost:8080/api/v1/merchant/products/<product-id>/variants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -d '{
    "sku": "TEST-SKU-003",
    "title": "Large",
    "price_cents": 100000,
    "compare_price_cents": 120000,
    "currency": "ETB",
    "attributes": {"size": "L"},
    "is_active": true
  }'
```

#### 6f. Update Product Variant Inventory

First, let's note that inventory is managed separately. Check product handlers for inventory endpoints.

---

### Phase 7: Admin Operations

#### 7a. List Users

```bash
curl http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <admin-access-token>"
```

#### 7b. Get User by ID

```bash
curl http://localhost:8080/api/v1/admin/users/00000000-0000-0000-0000-000000000002 \
  -H "Authorization: Bearer <admin-access-token>"
```

#### 7c. Update User KYC Status

```bash
curl -X PUT http://localhost:8080/api/v1/admin/users/00000000-0000-0000-0000-000000000002/kyc \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-access-token>" \
  -d '{"kyc_status":"verified"}'
```

#### 7d. List Merchants

```bash
curl http://localhost:8080/api/v1/admin/merchants \
  -H "Authorization: Bearer <admin-access-token>"
```

#### 7e. Verify Merchant

```bash
curl -X PUT http://localhost:8080/api/v1/admin/merchants/20000000-0000-0000-0000-000000000001/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-access-token>" \
  -d '{"is_verified":true}'
```

#### 7f. List Payouts (Admin)

```bash
curl http://localhost:8080/api/v1/admin/payouts \
  -H "Authorization: Bearer <admin-access-token>"
```

#### 7g. Process Payout (Admin)

```bash
curl -X POST http://localhost:8080/api/v1/admin/payouts/<payout-id>/process \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-access-token>" \
  -d '{
    "status": "completed",
    "reference": "BANK-REF-123"
  }'
```

---

### Phase 8: Checkout Flow

#### 8a. Create Address (if not exists)

```bash
curl -X POST http://localhost:8080/api/v1/addresses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -d '{
    "label": "Delivery",
    "address_line1": "Bole Sub-City",
    "city": "Addis Ababa",
    "state": "Addis Ababa",
    "postal_code": "1000",
    "country": "Ethiopia",
    "is_default": true
  }'
```

#### 8b. Add Item to Cart

```bash
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -d '{"variant_id":"50000000-0000-0000-0000-000000000001","quantity":1}'
```

#### 8c. Initiate Checkout

**Note:** The checkout endpoint requires an `Idempotency-Key` header to prevent duplicate payments. Use a unique value for each checkout attempt.

```bash
curl -X POST http://localhost:8080/api/v1/orders/checkout \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <buyer-access-token>" \
  -H "Idempotency-Key: <unique-idempotency-key>" \
  -d '{
    "address_id": "<address-id>",
    "payment_method": "chapa",
    "currency": "ETB"
  }'
```

**Idempotency Key Guidelines:**
- Use a unique UUID or random string for each checkout
- If the request is retried with the same key, the same response is returned without creating a duplicate order
- Cached responses are stored for 24 hours
- Response header `X-Idempotent-Replayed: true` indicates a replayed request

**Expected Response:**
```json
{
  "order_id": "uuid",
  "checkout_url": "https://checkout.chapa.co/...",
  "tx_ref": "tx_ref_..."
}
```

---

### Phase 9: Order Management

#### 9a. List My Orders (Buyer)

```bash
curl http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 9b. Get Order Details (Buyer)

```bash
curl http://localhost:8080/api/v1/orders/<order-id> \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 9c. Cancel Order

```bash
curl -X POST http://localhost:8080/api/v1/orders/<order-id>/cancel \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 9d. Confirm Delivery

```bash
curl -X POST http://localhost:8080/api/v1/orders/<order-id>/confirm-delivery \
  -H "Authorization: Bearer <buyer-access-token>"
```

#### 9e. List Merchant Orders

```bash
curl http://localhost:8080/api/v1/merchant/orders \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 9f. Get Merchant Order Details

```bash
curl http://localhost:8080/api/v1/merchant/orders/<order-id> \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 9g. Update Order Status

```bash
curl -X PUT http://localhost:8080/api/v1/merchant/orders/<order-id>/status \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -d '{"status":"processing"}'
```

#### 9h. Mark Order as Shipped

```bash
curl -X PUT http://localhost:8080/api/v1/merchant/orders/<order-id>/ship \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -d '{"tracking_number":"TRACK123456"}'
```

---

### Phase 10: Wallet & Payouts

#### 10a. Get Merchant Wallet Balance

```bash
curl http://localhost:8080/api/v1/merchant/wallet \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 10b. List Wallet Transactions

```bash
curl http://localhost:8080/api/v1/merchant/wallet/transactions \
  -H "Authorization: Bearer <merchant-access-token>"
```

#### 10c. Request Payout

**Note:** The payout endpoint requires an `Idempotency-Key` header to prevent duplicate payout requests.

```bash
curl -X POST http://localhost:8080/api/v1/merchant/wallet/payout \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <merchant-access-token>" \
  -H "Idempotency-Key: <unique-idempotency-key>" \
  -d '{
    "amount_cents": 50000,
    "currency": "ETB"
  }'
```

#### 10d. List Merchant Payouts

```bash
curl http://localhost:8080/api/v1/merchant/payouts \
  -H "Authorization: Bearer <merchant-access-token>"
```

---

## Webhook Testing

### Simulate Chapa Webhook

```bash
curl -X POST http://localhost:8080/webhooks/chapa \
  -H "Content-Type: application/json" \
  -d '{
    "tx_ref": "tx_ref_123",
    "status": "success",
    "amount": "500.00",
    "currency": "ETB"
  }'
```

Note: Real webhooks from Chapa are signed. This test webhook may be rejected without proper signature verification.

---

## Troubleshooting

### Common Issues

1. **401 Unauthorized**
   - Check token is valid and not expired
   - Ensure `Authorization: Bearer <token>` format

2. **403 Forbidden**
   - User may not have required role (e.g., admin endpoints)
   - Merchant endpoints require a merchant profile

3. **404 Not Found**
   - Check URL path and parameters
   - Verify resource exists (product slug, order ID, etc.)

4. **500 Internal Server Error**
   - Check server logs: `docker compose -f docker-compose-dev.yml logs app`
   - Verify database connectivity
   - Check Blnk service is running

5. **Database Connection Error**
   - Ensure Postgres is running: `docker compose -f docker-compose-dev.yml ps`
   - Check connection string in `.env`

### Reset Database

```bash
make migrate-down
make migrate-up
make seed-dev
```

### Check Service Health

```bash
# Docker services
docker compose -f docker-compose-dev.yml ps

# API health
curl http://localhost:8080/health

# Database connection
make psql
# Then: \dt to list tables
```

### View Logs

```bash
# All services
docker compose -f docker-compose-dev.yml logs -f

# App only
docker compose -f docker-compose-dev.yml logs -f app

# Postgres
docker compose -f docker-compose-dev.yml logs -f postgres
```

---

## API Documentation

Swagger UI is available at: http://localhost:8080/swagger/index.html

Generate docs:
```bash
make swagger
```