# Gebeya Marketplace Platform - User Flows

> This document describes the complete user flows for all user roles in the Gebeya marketplace platform, based on the actual implementation.

---

## Table of Contents

1. [Authentication Flows](#1-authentication-flows)
2. [Buyer Flows](#2-buyer-flows)
3. [Merchant Flows](#3-merchant-flows)
4. [Admin Flows](#4-admin-flows)
5. [System/Internal Flows](#5-systeminternal-flows)
6. [Order Status Lifecycle](#6-order-status-lifecycle)
7. [Payment & Escrow Flow](#7-payment--escrow-flow)

---

## 1. Authentication Flows

### 1.1 User Registration

**Endpoint:** `POST /api/v1/register`

**Access:** Public (no authentication required)

**Request Body:**
```json
{
  "email": "user@example.com",      // optional if phone provided
  "phone": "+251911234567",         // optional if email provided
  "password": "securepassword123",
  "role": "buyer"                   // optional, defaults to "buyer" | can be "merchant"
}
```

**Flow:**
```
Client → API: POST /register with email/phone, password, role
    ↓
AuthService.Register()
    ↓
Validate input (email OR phone required, password min 8 chars)
    ↓
Check if email/phone already exists
    ↓
Hash password with Argon2id
    ↓
Create User record with:
  - role: "buyer" or "merchant"
  - kyc_status: "pending"
  - is_active: true
    ↓
Generate JWT access token (15 min expiry)
    ↓
Generate refresh token (7 day expiry, stored in Redis)
    ↓
Return: { user, access_token, refresh_token }
```

**Response:** `201 Created`
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "phone": "+251911234567",
    "role": "buyer",
    "kyc_status": "pending",
    "created_at": "2026-03-26T..."
  },
  "token": {
    "access_token": "eyJhbGciOiJSUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
    "expires_in": 900
  }
}
```

### 1.2 User Login

**Endpoint:** `POST /api/v1/login`

**Access:** Public

**Request Body:**
```json
{
  "identifier": "user@example.com",  // email or phone
  "password": "securepassword123"
}
```

**Flow:**
```
Client → API: POST /login with identifier and password
    ↓
AuthService.Login()
    ↓
Lookup user by email OR phone
    ↓
Verify password hash
    ↓
Generate access token and refresh token
    ↓
Return: { user, access_token, refresh_token }
```

### 1.3 Token Refresh

**Endpoint:** `POST /api/v1/refresh`

**Access:** Public

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
}
```

**Flow:**
```
Client → API: POST /refresh with refresh_token
    ↓
AuthService.Refresh()
    ↓
Validate refresh token exists in Redis
    ↓
Generate new access token and refresh token
    ↓
Delete old refresh token, store new one
    ↓
Return: { access_token, refresh_token, expires_in }
```

### 1.4 Logout

**Endpoint:** `POST /api/v1/logout`

**Access:** Public (requires refresh_token)

**Flow:**
```
Client → API: POST /logout with refresh_token
    ↓
AuthService.Logout()
    ↓
Delete refresh token from Redis
    ↓
Return: 204 No Content
```

### 1.5 Logout All Sessions

**Endpoint:** `POST /api/v1/logout-all`

**Access:** Authenticated

**Flow:**
```
Client → API: POST /logout-all (with Bearer token)
    ↓
AuthService.LogoutAll()
    ↓
Delete ALL refresh tokens for the user from Redis
    ↓
Return: 204 No Content
```

---

## 2. Buyer Flows

### 2.1 Browse Products (Public)

**Endpoints:**
- `GET /api/v1/products` - List products (paginated)
- `GET /api/v1/products/{slug}` - Get product by slug
- `GET /api/v1/categories` - List categories
- `GET /api/v1/categories/{id}` - Get category
- `GET /api/v1/shops/{shop_slug}` - Get shop info

**Access:** Public

**Flow:**
```
Client → API: GET /products?page=1&per_page=20
    ↓
ProductService.ListProducts()
    ↓
Query products with variants, images, inventory
    ↓
Return: { items: [...], total, page, per_page, total_pages }
```

### 2.2 Cart Management

#### 2.2.1 Get Cart (Guest or Authenticated)

**Endpoint:** `GET /api/v1/cart`

**Access:** Session-based (works for both guests and authenticated users)

**Flow:**
```
Client → API: GET /cart (with X-Session-ID header or cookie)
    ↓
Session Middleware:
    - Extract or create session_id
    - Store in context
    ↓
If authenticated:
    CartService.GetOrCreateCart(userID, null)
Else:
    CartService.GetOrCreateCart(null, sessionID)
    ↓
Return: { cart_id, items: [...], total_cents }
```

#### 2.2.2 Add Item to Cart

**Endpoint:** `POST /api/v1/cart/items`

**Access:** Session-based

**Request Body:**
```json
{
  "variant_id": "uuid",
  "quantity": 2
}
```

**Flow:**
```
Client → API: POST /cart/items
    ↓
CartService.AddItem()
    ↓
Validate variant exists and is active
    ↓
Get current price from variant
    ↓
Check inventory availability:
    - available = quantity - reserved
    - If available < quantity: return error
    ↓
Reserve inventory: UPDATE inventory SET reserved = reserved + quantity
    ↓
Create or update cart item
    ↓
Return: { cart_item }
```

#### 2.2.3 Update Item Quantity

**Endpoint:** `PUT /api/v1/cart/items/{variantId}`

**Request Body:**
```json
{
  "quantity": 3
}
```

**Flow:**
```
Client → API: PUT /cart/items/{variantId}
    ↓
If quantity > current:
    CartService.AddItem() // reserves additional inventory
Else if quantity < current:
    CartService.RemoveItem() // releases difference
Else:
    No change
```

#### 2.2.4 Remove Item from Cart

**Endpoint:** `DELETE /api/v1/cart/items/{variantId}`

**Flow:**
```
Client → API: DELETE /cart/items/{variantId}
    ↓
CartService.RemoveItemByVariant()
    ↓
Release inventory reservation
    ↓
Delete cart item
    ↓
Return: 204 No Content
```

#### 2.2.5 Merge Guest Cart (After Login)

**Endpoint:** `POST /api/v1/cart/merge`

**Access:** Authenticated + Session

**Flow:**
```
Client → API: POST /cart/merge (with Bearer token and session)
    ↓
CartService.MergeCart(userID, sessionID)
    ↓
Get guest cart by session_id
Get user cart by user_id
    ↓
For each item in guest cart:
    If variant exists in user cart:
        Take higher quantity
    Else:
        Insert item into user cart
    ↓
Delete guest cart
    ↓
Return: { merged_cart }
```

### 2.3 Address Management

**Endpoints:**
- `GET /api/v1/addresses` - List user's addresses
- `POST /api/v1/addresses` - Create address
- `GET /api/v1/addresses/{id}` - Get address
- `PUT /api/v1/addresses/{id}` - Update address
- `DELETE /api/v1/addresses/{id}` - Delete address
- `PUT /api/v1/addresses/{id}/default` - Set default address

**Access:** Authenticated

**Request Body (Create/Update):**
```json
{
  "full_name": "Abebe Bikila",
  "phone": "+251911234567",
  "line1": "Bole Road, Building 123",
  "line2": "Apartment 4B",
  "city": "Addis Ababa",
  "subcity": "Bole",
  "woreda": "03",
  "is_default": true
}
```

### 2.4 Checkout Flow

**Endpoint:** `POST /api/v1/orders/checkout`

**Access:** Authenticated + Session

**Request Body:**
```json
{
  "shipping_address_id": "uuid",  // optional if only one address
  "notes": "Please leave at the door"
}
```

**Complete Flow:**
```
Client → API: POST /orders/checkout
    ↓
CheckoutService.Checkout()
    ↓
══════════════════════════════════════════════════════════════
PHASE 1: VALIDATION & PREPARATION
══════════════════════════════════════════════════════════════
    ↓
Get user's cart (by userID or sessionID)
    ↓
Validate cart is not empty
    ↓
If authenticated:
    Use provided shipping_address_id or default address
If guest:
    Require shipping_address_id in request
    ↓
For each cart item:
    - Verify variant exists and is active
    - Verify inventory available (quantity - reserved >= item.quantity)
    - Snapshot product name, variant title, unit_price
    ↓
Calculate:
    - subtotal_cents = sum(item.unit_price × item.quantity)
    - platform_fee_cents = subtotal_cents × 5%
    - total_cents = subtotal_cents (buyer pays subtotal, fee deducted from merchant)
    ↓
Generate unique chapa_tx_ref: "GEBEYA-{orderID}-{timestamp}"
    ↓
══════════════════════════════════════════════════════════════
PHASE 2: DATABASE TRANSACTION (BEGIN)
══════════════════════════════════════════════════════════════
    ↓
Create Order record:
    - status: "pending_payment"
    - buyer_id: userID
    - merchant_id: (from first cart item's product)
    - subtotal_cents, platform_fee_cents, total_cents
    - chapa_tx_ref
    ↓
Create Order Items (snapshot of product data)
    ↓
Create Payment record:
    - status: "pending"
    - amount_cents, currency: "ETB"
    - provider: "chapa"
    - provider_tx_ref: chapa_tx_ref
    ↓
Convert cart to order (mark cart as converted)
    ↓
Enqueue cart expiry job (24 hours to complete payment)
    ↓
══════════════════════════════════════════════════════════════
PHASE 3: EXTERNAL SERVICES (OUTSIDE DB TRANSACTION)
══════════════════════════════════════════════════════════════
    ↓
Initialize Chapa Payment:
    POST https://api.chapa.co/v1/transaction/initialize
    {
        "amount": "2650.00",          // total in ETB
        "currency": "ETB",
        "email": "buyer@example.com",
        "first_name": "Abebe",
        "last_name": "Bikila",
        "phone_number": "+251911234567",
        "tx_ref": "GEBEYA-uuid-1234567890",
        "callback_url": "https://api.example.com/webhooks/chapa",
        "return_url": "https://app.example.com/payment/complete",
        "customization[title]": "Gebeya Purchase",
        "customization[description]": "Order #uuid"
    }
    ↓
If Chapa fails:
    - Void Blnk escrow (if created)
    - Release inventory reservations
    - Mark order as cancelled
    - Return error
    ↓
Create Blnk Escrow Hold (Inflight Transaction):
    BlnkClient.CreateEscrowHold({
        order_id: orderID,
        merchant_id: merchantID,
        amount_cents: total_cents
    })
    - Creates inflight transaction in Blnk
    - Funds held in escrow balance
    - Not yet committed (can be voided)
    ↓
═ Commit DB Transaction (All succeed or all fail) ═
    ↓
══════════════════════════════════════════════════════════════
PHASE 4: RESPONSE TO CLIENT
══════════════════════════════════════════════════════════════
    ↓
Return: {
    "order_id": "uuid",
    "total_cents": 265000,
    "currency": "ETB",
    "payment_url": "https://checkout.chapa.co/checkout/payment/xxxxx",
    "chapa_tx_ref": "GEBEYA-uuid-1234567890"
}
```

**Client Action:** Redirect buyer to `payment_url` to complete payment.

### 2.5 View Orders

**Endpoints:**
- `GET /api/v1/orders` - List buyer's orders
- `GET /api/v1/orders/{id}` - Get order details

**Access:** Authenticated

**Response includes:** Order info, items, and shipment details (if shipped).

### 2.6 Confirm Delivery

**Endpoint:** `POST /api/v1/orders/{id}/confirm-delivery`

**Access:** Authenticated (must be the buyer of the order)

**Pre-condition:** Order status must be `shipped`

**Flow:**
```
Client → API: POST /orders/{id}/confirm-delivery
    ↓
OrderService.ConfirmDelivery()
    ↓
Validate order belongs to buyer
Validate order status is "shipped"
    ↓
Update order status → "delivered"
Update shipment.delivered_at → now()
    ↓
Commit Blnk Escrow:
    BlnkClient.CommitEscrow(inflight_txn_id)
    ↓
Credit Merchant Wallet:
    merchant_payout = subtotal - platform_fee
    BlnkClient.CreditMerchant({
        order_id: orderID,
        merchant_id: merchantID,
        subtotal_cents: subtotal,
        fee_cents: platform_fee,
        merchant_blnk_id: merchant.blnk_balance_id
    })
    ↓
Create Wallet Transaction:
    - type: "sale_credit"
    - amount_cents: merchant_payout
    - status: "applied"
    - blnk_transaction_id: from CreditMerchant
    ↓
Release inventory commit (already done on payment success)
    ↓
Return: { order with status "delivered" }
```

---

## 3. Merchant Flows

### 3.1 Become a Merchant (Onboarding)

**Endpoint:** `POST /api/v1/merchant/onboard`

**Access:** Authenticated (any role can onboard to become a merchant)

**Request Body:**
```json
{
  "shop_name": "Tigist Habesha Designs",
  "shop_slug": "tigist-habesha",
  "shop_description": "Traditional Ethiopian clothing",
  "contact_phone": "+251911234567",
  "contact_email": "tigist@example.com",
  "address": "Bole, Addis Ababa",
  "city": "Addis Ababa"
}
```

**Flow:**
```
Client → API: POST /merchant/onboard
    ↓
MerchantService.Onboard()
    ↓
Validate user doesn't already have a merchant profile
    ↓
If shop_slug not provided:
    Generate from shop_name (lowercase, hyphens)
    ↓
Check shop_slug is unique
    ↓
Create Blnk Wallet:
    BlnkClient.CreateMerchantWallet(merchantID, shopName)
    - Creates balance in Blnk ledger
    - Returns: blnk_balance_id
    ↓
Create Merchant Profile record:
    - user_id: from JWT
    - shop_name, shop_slug, contact_phone, city
    - blnk_balance_id: from Blnk
    - is_verified: false
    - total_sales_cents: 0
    ↓
Return: {
    "merchant_id": "uuid",
    "shop_slug": "tigist-habesha",
    "shop_name": "Tigist Habesha Designs",
    "message": "Merchant profile created. KYC verification pending."
}
```

**Note:** User's role in `users` table remains unchanged. The `RequireMerchantProfile` middleware checks for existence of a merchant profile, not the role.

### 3.2 Update Merchant Profile

**Endpoint:** `PUT /api/v1/merchant/profile`

**Access:** Requires Merchant Profile

### 3.3 Manage Products

**Endpoints:**
- `GET /api/v1/merchant/products` - List merchant's products
- `POST /api/v1/merchant/products` - Create product
- `PUT /api/v1/merchant/products/{id}` - Update product
- `DELETE /api/v1/merchant/products/{id}` - Delete product
- `POST /api/v1/merchant/products/{id}/variants` - Add variant
- `PUT /api/v1/merchant/products/{id}/variants/{variantId}` - Update variant
- `DELETE /api/v1/merchant/products/{id}/variants/{variantId}` - Delete variant

**Access:** Requires Merchant Profile

**Create Product Request:**
```json
{
  "category_id": "uuid",
  "name": "Traditional Habesha Kemis",
  "slug": "traditional-habesha-kemis",
  "description": "Handwoven Ethiopian traditional dress",
  "status": "active",      // "draft" or "active"
  "is_digital": false
}
```

**Product Status Flow:**
```
draft → active → archived
```

### 3.4 Manage Orders

**Endpoints:**
- `GET /api/v1/merchant/orders` - List merchant's orders
- `GET /api/v1/merchant/orders/{id}` - Get order details with shipment
- `PUT /api/v1/merchant/orders/{id}/status` - Update order status
- `PUT /api/v1/merchant/orders/{id}/ship` - Mark order as shipped

**Access:** Requires Merchant Profile

#### Update Order Status

**Request:**
```json
{
  "status": "processing"  // "paid", "processing", "shipped", "cancelled"
}
```

**Valid Transitions:**
```
paid → processing → shipped → (delivered by buyer)
paid → cancelled
```

**Flow for `processing` status:**
```
Merchant → API: PUT /merchant/orders/{id}/status {status: "processing"}
    ↓
Validate order belongs to this merchant
Validate status transition is valid (paid → processing)
    ↓
Update order status → "processing"
    ↓
Create Shipment record:
    - order_id: orderID
    - status: "preparing"
    ↓
Return: { order }
```

#### Ship Order

**Endpoint:** `PUT /api/v1/merchant/orders/{id}/ship`

**Request:**
```json
{
  "carrier": "Ethiopian Post",
  "tracking_number": "ET123456789",
  "estimated_delivery": "2026-04-01"
}
```

**Pre-condition:** Order status must be `processing`

**Flow:**
```
Merchant → API: PUT /merchant/orders/{id}/ship
    ↓
Validate order belongs to this merchant
Validate order status is "processing"
    ↓
Update order status → "shipped"
Update shipment:
    - status → "shipped"
    - carrier: provided
    - tracking_number: provided
    - estimated_delivery: provided
    - shipped_at: now()
    ↓
Enqueue SendOrderNotificationWorker (async notification)
    ↓
Return: { order, shipment }
```

### 3.5 Wallet & Payouts

**Endpoints:**
- `GET /api/v1/merchant/wallet` - Get wallet balance
- `GET /api/v1/merchant/wallet/transactions` - List transaction history
- `POST /api/v1/merchant/wallet/payout` - Request payout

**Access:** Requires Merchant Profile

#### Get Wallet Balance

**Flow:**
```
Merchant → API: GET /merchant/wallet
    ↓
WalletService.GetBalance()
    ↓
Get merchant profile (includes blnk_balance_id)
    ↓
If no blnk_balance_id:
    Return { balance_cents: 0, currency: "ETB" }
    ↓
Query Blnk API:
    BlnkClient.GetBalance(merchant.blnk_balance_id)
    ↓
Return: {
    "balance_cents": 150000,
    "currency": "ETB"
}
```

#### Request Payout

**Request:**
```json
{
  "amount_cents": 50000,
  "method": "telebirr",         // "telebirr", "bank_transfer", "cbe_birr"
  "destination": "+251911234567"
}
```

**Flow:**
```
Merchant → API: POST /merchant/wallet/payout
    ↓
WalletService.RequestPayout()
    ↓
Get merchant profile with blnk_balance_id
    ↓
Query Blnk for current balance
    ↓
Validate balance >= requested amount
Validate amount >= minimum (1 ETB = 100 cents)
    ↓
Create Payout record:
    - merchant_id: merchantID
    - amount_cents: requested
    - currency: "ETB"
    - method: "telebirr"
    - destination: provided
    - status: "pending"
    ↓
Return: {
    "id": "payout_uuid",
    "status": "pending",
    "amount_cents": 50000,
    ...
}
    ↓
Note: Actual processing happens asynchronously via admin
```

---

## 4. Admin Flows

### 4.1 User Management

**Endpoints:**
- `GET /api/v1/admin/users` - List all users
- `GET /api/v1/admin/users/{id}` - Get user
- `DELETE /api/v1/admin/users/{id}` - Delete user

**Access:** Requires `admin` role

### 4.2 Category Management

**Endpoints:**
- `POST /api/v1/admin/categories` - Create category
- `PUT /api/v1/admin/categories/{id}` - Update category
- `DELETE /api/v1/admin/categories/{id}` - Delete category

**Access:** Requires `admin` role

### 4.3 Payout Management

**Endpoints:**
- `GET /api/v1/admin/payouts` - List payouts (with status filter)
- `POST /api/v1/admin/payouts/{id}/process` - Process payout

**List Payouts:**
```
GET /api/v1/admin/payouts?status=pending&page=1&per_page=20
```

**Process Payout Flow:**
```
Admin → API: POST /admin/payouts/{id}/process
    ↓
Validate payout exists and status is "pending"
    ↓
Update payout status → "processing"
    ↓
Return: {
    "message": "Payout processing initiated...",
    "payout_id": "uuid"
}
    ↓
ProcessPayoutWorker (async):
    ↓
    Get payout & merchant records
    ↓
    Validate status is "processing"
    ↓
    Get merchant's Blnk balance
    ↓
    Validate sufficient balance
    ↓
    Debit from Blnk wallet:
        BlnkClient.DebitMerchantForPayout({
            payout_id: payoutID,
            merchant_id: merchantID,
            amount_cents: payout.amount_cents,
            method: payout.method,
            destination: payout.destination,
            merchant_blnk_id: merchant.blnk_balance_id
        })
    ↓
    Update payout status → "completed"
    Set blnk_transaction_id
    Set processed_at → now()
    ↓
    (Or on failure:)
    Update payout status → "failed"
    Set failure_reason
```

### 4.4 KYC & Verification (To Be Implemented)

**Endpoints:**
- `PUT /api/v1/admin/users/{id}/kyc` - Update KYC status ⚠️ *Not implemented*
- `PUT /api/v1/admin/admin/merchants/{id}/verify` - Verify merchant ⚠️ *Not implemented*

---

## 5. System/Internal Flows

### 5.1 Payment Webhook (Chapa → Gebeya)

**Endpoint:** `POST /webhooks/chapa`

**Access:** Public (validated by signature)

**Flow:**
```
Chapa → API: POST /webhooks/chapa
    Headers:
        Chapa-Signature: <hmac_signature>
    Body:
        {
            "event": "charge.success",
            "data": {
                "tx_ref": "GEBEYA-uuid-1234567890",
                "amount": "2650.00",
                "currency": "ETB",
                "status": "success"
            }
        }
    ↓
WebhookHandler.HandleChapa()
    ↓
Verify Chapa-Signature header
    ↓
Parse webhook event
    ↓
Find order by chapa_tx_ref
    ↓
Enqueue ProcessPaymentWebhookWorker:
    { tx_ref: "GEBEYA-...", status: "success" }
    ↓
Return: 200 OK { "received": true }
    ↓
ProcessPaymentWebhookWorker (async):
    ↓
    Get order by chapa_tx_ref
    ↓
    If status == "success":
        Update payment → status: "succeeded"
        Update order → status: "paid"
        Commit inventory reservations:
            For each order item:
                inventory.Commit(variant_id, quantity)
        
    Else if status == "failed":
        Update payment → status: "failed"
        Update order → status: "cancelled"
        Void Blnk escrow (if exists)
        Release inventory reservations:
            For each order item:
                inventory.Release(variant_id, quantity)
```

### 5.2 Cart Expiry (Background Job)

**Job:** `ReleaseCartReservationWorker`

**Trigger:** Scheduled 24 hours after checkout

**Flow:**
```
Cart expires (24 hours after creation without checkout)
    ↓
ReleaseCartReservationWorker.Execute({ cart_id: uuid })
    ↓
Get cart by ID
    ↓
If cart is still "active" (not converted to order):
    Get all cart items
    ↓
    For each item:
        Release inventory reservation:
            inventory.Release(variant_id, quantity)
    ↓
    Mark cart as "expired"
```

### 5.3 Order Notification (Background Job)

**Job:** `SendOrderNotificationWorker`

**Trigger:** On order status changes

**Flow:**
```
Order status changes (e.g., shipped)
    ↓
SendOrderNotificationWorker.Execute({
    order_id: uuid,
    event: "shipped",
    phone: "+251911234567"
})
    ↓
Send SMS via Africa's Talking (or similar):
    "Your order #123 has been shipped. Track: ET123456789"
```

### 5.4 Payout Processing (Background Job)

**Job:** `ProcessPayoutWorker`

**Trigger:** Admin initiates payout processing

**See:** Section 4.3 Process Payout Flow

---

## 6. Order Status Lifecycle

### State Machine

```
                    ┌──────────────────────┐
                    │                      │
                    ▼                      │
            ┌───────────────┐        ┌────────────┐
            │ pending_payment│───────▶│ cancelled  │
            └───────┬───────┘        └────────────┘
                    │                      │
                    │ payment success     │ payment fail
                    ▼                      │
            ┌───────────────┐              │
            │     paid     │              │
            └───────┬───────┘              │
                    │                      │
                    │ merchant processes   │
                    ▼                      │
            ┌───────────────┐              │
            │  processing   │              │
            └───────┬───────┘              │
                    │                      │
                    │ merchant ships       │
                    ▼                      │
            ┌───────────────┐              │
            │   shipped     │              │
            └───────┬───────┘              │
                    │                      │
                    │ buyer confirms       │
                    ▼                      │
            ┌───────────────┐              │
            │  delivered    │              │
            └───────────────┘              │
                                           │
                    ┌──────────────────────┘
                    │
                    ▼
            ┌───────────────┐
            │   refunded    │ (manual admin action)
            └───────────────┘
```

### Status Actions

| Status | What Happens | Who Can Change |
|--------|--------------|----------------|
| `pending_payment` | Order created, inventory reserved, escrow created | System (checkout) |
| `paid` | Payment successful, inventory committed | System (webhook) |
| `processing` | Shipment record created | Merchant |
| `shipped` | Carrier & tracking set, notification sent | Merchant |
| `delivered` | Escrow committed, merchant wallet credited | Buyer (confirmation) |
| `cancelled` | Inventory released, escrow voided | System (payment failure) |
| `refunded` | Manual process | Admin (not implemented) |

---

## 7. Payment & Escrow Flow

### Payment Flow (Chapa)

```
┌─────────┐      ┌─────────┐      ┌─────────┐      ┌─────────┐
│  Buyer  │      │ Gebeya  │      │  Chapa  │      │  Blnk   │
└────┬────┘      └────┬────┘      └────┬────┘      └────┬────┘
     │                │                │                 │
     │ 1. Checkout    │                │                 │
     │───────────────▶│                │                 │
     │                │ 2. Initialize  │                 │
     │                │   Payment      │                 │
     │                │───────────────▶│                 │
     │                │◀───────────────│                 │
     │                │  checkout_url  │                 │
     │◀───────────────│                │                 │
     │ redirect_url  │                │                 │
     │                │                │                 │
     │ 3. Pay on      │                │                 │
     │    Chapa page  │                │                 │
     │────────────────────────────────▶│                 │
     │                │                │                 │
     │                │ 4. Webhook     │                 │
     │                │◀───────────────│                 │
     │                │                │                 │
     │                │ 5. Commit      │                 │
     │                │   Escrow       │                 │
     │                │─────────────────────────────────▶│
     │                │                │                 │
     │                │ 6. Credit      │                 │
     │                │   Merchant     │                 │
     │                │                │                 │
     │◀───────────────│ (on delivery) │                 │
     │                │                │                 │
```

### Escrow Hold (Pre-Delivery)

When checkout completes:
- Blnk creates an **inflight transaction** (not yet committed)
- Funds are held in platform's escrow balance
- Order shows `blnk_transaction_id`

### Escrow Commit (On Delivery)

When buyer confirms delivery:
1. `CommitEscrow()` - Marks inflight transaction as committed
2. `CreditMerchant()` - Transfers (subtotal - fee) to merchant wallet
3. Remaining amount (fee) stays in revenue balance

---

## Appendix: Role & Permission Matrix

| Permission | Buyer | Merchant | Admin |
|------------|-------|----------|-------|
| Browse products | ✅ | ✅ | ✅ |
| Manage cart | ✅ | ✅ | ✅ |
| Place orders | ✅ | ✅ | ✅ |
| View own orders | ✅ | ✅ | ✅ |
| Become merchant | ✅ | - | - |
| Manage products | - | ✅ | - |
| View merchant orders | - | ✅ | - |
| View wallet | - | ✅ | - |
| Request payout | - | ✅ | - |
| Manage all users | - | - | ✅ |
| Manage categories | - | - | ✅ |
| Process payouts | - | - | ✅ |
| Verify KYC | - | - | ⚠️ |
| Verify merchants | - | - | ⚠️ |

⚠️ = Endpoint exists but not implemented

---

## Appendix: API Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `bad_request` | Invalid request body or parameters |
| 401 | `unauthorized` | Missing or invalid authentication |
| 403 | `forbidden` | Insufficient permissions |
| 404 | `not_found` | Resource not found |
| 409 | `conflict` | Resource already exists |
| 422 | `validation_error` | Request validation failed |
| 500 | `internal_error` | Server error |

---

*Last updated: 2026-03-26*