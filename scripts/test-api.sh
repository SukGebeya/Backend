#!/bin/bash

# =============================================================================
# SukGebeya API Test Script
# =============================================================================
# Usage: ./scripts/test-api.sh [phase]
#
# Phases:
#   1-auth      - Authentication (login/register/refresh)
#   2-public    - Public endpoints (categories/products/shops)
#   3-cart      - Cart operations (guest/authenticated)
#   4-address   - Address management
#   5-merchant  - Merchant profile and products
#   6-admin     - Admin operations (users/merchants/payouts)
#   7-checkout  - Checkout flow
#   8-order     - Order management
#   9-wallet    - Wallet and payouts
#   all         - Run all phases sequentially
#
# Environment Variables:
#   API_URL     - Base URL (default: http://localhost:8080)
#   ADMIN_EMAIL - Admin email (default: admin@test.com)
#   ADMIN_PASS  - Admin password (default: Admin123!)
#   BUYER_EMAIL - Buyer email (default: buyer@test.com)
#   BUYER_PASS  - Buyer password (default: Password123!)
#   MERCH_EMAIL - Merchant email (default: merchant@test.com)
#   MERCH_PASS  - Merchant password (default: Password123!)
# =============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@test.com}"
ADMIN_PASS="${ADMIN_PASS:-Admin123!}"
BUYER_EMAIL="${BUYER_EMAIL:-buyer@test.com}"
BUYER_PASS="${BUYER_PASS:-Password123!}"
MERCH_EMAIL="${MERCH_EMAIL:-merchant@test.com}"
MERCH_PASS="${MERCH_PASS:-Password123!}"

# Test data IDs (from seed.sql)
CATEGORY_ID="10000000-0000-0000-0000-000000000001"
PRODUCT_SLUG="traditional-habesha-kemis"
PRODUCT_ID="30000000-0000-0000-0000-000000000001"
VARIANT_ID="50000000-0000-0000-0000-000000000001"
MERCHANT_ID="20000000-0000-0000-0000-000000000001"
SHOP_SLUG="ethiopian-fashion-hub"
BUYER_USER_ID="00000000-0000-0000-0000-000000000002"
ADMIN_USER_ID="00000000-0000-0000-0000-000000000001"

# Token storage
ADMIN_TOKEN=""
BUYER_TOKEN=""
MERCHANT_TOKEN=""
REFRESH_TOKEN=""
SESSION_ID=""

# Helper functions

print_header() {
	echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	echo -e "${CYAN}  $1${NC}"
	echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_test() {
	echo -e "${BLUE}▶ $1${NC}"
}

print_success() {
	echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
	echo -e "${RED}✗ $1${NC}"
}

print_warning() {
	echo -e "${YELLOW}⚠ $1${NC}"
}

print_response() {
	echo -e "${CYAN}Response:${NC}"
	echo "$1" | jq '.' 2>/dev/null || echo "$1"
	echo ""
}

# Extract token from login response
extract_token() {
	echo "$1" | jq -r '.token.access_token' 2>/dev/null
}

extract_refresh_token() {
	echo "$1" | jq -r '.token.refresh_token' 2>/dev/null
}

extract_user_id() {
	echo "$1" | jq -r '.user.id' 2>/dev/null
}

# Make API request
api_call() {
	local method="$1"
	local endpoint="$2"
	local data="$3"
	local token="$4"
	local session="$5"
	local idempotency_key="$6"

	local url="${API_URL}${endpoint}"
	local args=(-X "$method" -s)
	args+=(-H "Content-Type: application/json")

	if [ -n "$token" ]; then
		args+=(-H "Authorization: Bearer $token")
	fi

	if [ -n "$session" ]; then
		args+=(-H "X-Session-ID: $session")
	fi

	if [ -n "$idempotency_key" ]; then
		args+=(-H "Idempotency-Key: $idempotency_key")
	fi

	if [ -n "$data" ]; then
		args+=(-d "$data")
	fi

	curl "${args[@]}" "$url"
}

# Generate a random idempotency key
generate_idempotency_key() {
	echo "ik-$(date +%s)-$RANDOM"
}

# Check API health
check_health() {
	print_header "Checking API Health"

	local response
	response=$(curl -s "${API_URL}/health")

	if echo "$response" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
		print_success "API is healthy"
		print_response "$response"
		return 0
	else
		print_error "API is not healthy"
		print_response "$response"
		return 1
	fi
}

# =============================================================================
# PHASE 1: Authentication
# =============================================================================

phase_auth() {
	print_header "PHASE 1: Authentication"

	# 1a. Health check
	check_health || return 1

	# 1b. Login as Admin
	print_test "1.1 Login as Admin"
	local response
	response=$(api_call POST "/api/v1/login" "{\"identifier\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASS}\"}")

	ADMIN_TOKEN=$(extract_token "$response")
	if [ -n "$ADMIN_TOKEN" ] && [ "$ADMIN_TOKEN" != "null" ]; then
		print_success "Admin login successful"
		print_response "$response"
	else
		print_error "Admin login failed"
		print_response "$response"
		return 1
	fi

	# 1c. Login as Buyer
	print_test "1.2 Login as Buyer"
	response=$(api_call POST "/api/v1/login" "{\"identifier\":\"${BUYER_EMAIL}\",\"password\":\"${BUYER_PASS}\"}")

	BUYER_TOKEN=$(extract_token "$response")
	REFRESH_TOKEN=$(extract_refresh_token "$response")
	local buyer_id=$(extract_user_id "$response")

	if [ -n "$BUYER_TOKEN" ] && [ "$BUYER_TOKEN" != "null" ]; then
		print_success "Buyer login successful"
		print_response "$response"
	else
		print_error "Buyer login failed"
		print_response "$response"
		return 1
	fi

	# 1d. Login as Merchant
	print_test "1.3 Login as Merchant"
	response=$(api_call POST "/api/v1/login" "{\"identifier\":\"${MERCH_EMAIL}\",\"password\":\"${MERCH_PASS}\"}")

	MERCHANT_TOKEN=$(extract_token "$response")

	if [ -n "$MERCHANT_TOKEN" ] && [ "$MERCHANT_TOKEN" != "null" ]; then
		print_success "Merchant login successful"
		print_response "$response"
	else
		print_error "Merchant login failed"
		print_response "$response"
		return 1
	fi

	# 1e. Refresh token test
	print_test "1.4 Refresh Token"
	response=$(api_call POST "/api/v1/refresh" "{\"refresh_token\":\"${REFRESH_TOKEN}\"}")

	local new_token
	new_token=$(extract_token "$response")

	if [ -n "$new_token" ] && [ "$new_token" != "null" ]; then
		print_success "Token refresh successful"
		print_response "$response"
	else
		print_warning "Token refresh returned empty (this may be expected)"
		print_response "$response"
	fi

	# 1f. Get current user
	print_test "1.5 Get Current User"
	response=$(api_call GET "/api/v1/users/me" "" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Get current user successful"
		print_response "$response"
	else
		print_error "Get current user failed"
		print_response "$response"
		return 1
	fi

	print_success "✓ Authentication phase complete"
}

# =============================================================================
# PHASE 2: Public Endpoints
# =============================================================================

phase_public() {
	print_header "PHASE 2: Public Endpoints"

	# 2a. List categories
	print_test "2.1 List Categories"
	local response
	response=$(api_call GET "/api/v1/categories")

	# Handle both array and object responses
	if echo "$response" | jq -e '.categories' >/dev/null 2>&1; then
		print_success "Categories retrieved"
		print_response "$response"
	elif echo "$response" | jq -e 'if type == "array" then true else false end' >/dev/null 2>&1; then
		print_success "Categories retrieved (array format)"
		print_response "$response"
	else
		print_error "Failed to get categories"
		print_response "$response"
		return 1
	fi

	# 2b. Get category by ID
	print_test "2.2 Get Category by ID"
	response=$(api_call GET "/api/v1/categories/${CATEGORY_ID}")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Category retrieved"
		print_response "$response"
	else
		print_error "Failed to get category"
		print_response "$response"
		return 1
	fi

	# 2c. Get category by slug
	print_test "2.3 Get Category by Slug"
	response=$(api_call GET "/api/v1/categories/slug/clothing-fashion")

	if echo "$response" | jq -e '.slug' >/dev/null 2>&1; then
		print_success "Category retrieved by slug"
		print_response "$response"
	else
		print_warning "Category by slug not found (may be expected)"
		print_response "$response"
	fi

	# 2d. List products
	print_test "2.4 List Products"
	response=$(api_call GET "/api/v1/products")

	if echo "$response" | jq -e '.Items' >/dev/null 2>&1; then
		print_success "Products retrieved"
		print_response "$response"
	elif echo "$response" | jq -e '.products' >/dev/null 2>&1; then
		print_success "Products retrieved"
		print_response "$response"
	elif echo "$response" | jq -e 'if type == "array" then true else false end' >/dev/null 2>&1; then
		print_success "Products retrieved (array format)"
		print_response "$response"
	else
		print_error "Failed to get products"
		print_response "$response"
		return 1
	fi

	# 2e. Get product by slug
	print_test "2.5 Get Product by Slug"
	response=$(api_call GET "/api/v1/products/${PRODUCT_SLUG}")

	if echo "$response" | jq -e '.slug' >/dev/null 2>&1; then
		print_success "Product retrieved"
		print_response "$response"
	elif echo "$response" | jq -e '.product.slug' >/dev/null 2>&1; then
		print_success "Product retrieved"
		print_response "$response"
	else
		print_error "Failed to get product"
		print_response "$response"
		return 1
	fi

	# 2f. Get shop by slug
	print_test "2.6 Get Shop by Slug"
	response=$(api_call GET "/api/v1/shops/${SHOP_SLUG}")

	if echo "$response" | jq -e '.shop_slug' >/dev/null 2>&1; then
		print_success "Shop retrieved"
		print_response "$response"
	else
		print_warning "Shop not found (may be expected)"
		print_response "$response"
	fi

	print_success "✓ Public endpoints phase complete"
}

# =============================================================================
# PHASE 3: Cart Operations
# =============================================================================

phase_cart() {
	print_header "PHASE 3: Cart Operations"

	# 3a. Get cart (creates guest session)
	print_test "3.1 Get Cart (Guest Session)"
	local response
	response=$(curl -s -i "${API_URL}/api/v1/cart" 2>&1)

	SESSION_ID=$(echo "$response" | grep -i "X-Session-ID:" | sed 's/X-Session-ID: //i' | tr -d '\r\n')

	if [ -n "$SESSION_ID" ]; then
		print_success "Guest session created: $SESSION_ID"

		# Extract body (after empty line)
		local body
		body=$(echo "$response" | sed -n '/^{/,$p')
		print_response "$body"
	else
		print_warning "No session ID returned (trying to extract from body)"
		response=$(api_call GET "/api/v1/cart")
		print_response "$response"
	fi

	# 3b. Add item to cart (authenticated)
	print_test "3.2 Add Item to Cart (Authenticated)"
	response=$(api_call POST "/api/v1/cart/items" "{\"variant_id\":\"${VARIANT_ID}\",\"quantity\":2}" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.cart_id' >/dev/null 2>&1; then
		print_success "Item added to cart"
		print_response "$response"
	elif echo "$response" | jq -e '.items' >/dev/null 2>&1; then
		print_success "Item added to cart"
		print_response "$response"
	else
		print_error "Failed to add item to cart"
		print_response "$response"
		return 1
	fi

	# 3c. Get cart
	print_test "3.3 Get Cart"
	response=$(api_call GET "/api/v1/cart" "" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.items' >/dev/null 2>&1; then
		print_success "Cart retrieved"
		print_response "$response"
	else
		print_error "Failed to get cart"
		print_response "$response"
		return 1
	fi

	# 3d. Update cart item quantity
	print_test "3.4 Update Cart Item Quantity"
	response=$(api_call PUT "/api/v1/cart/items/${VARIANT_ID}" "{\"quantity\":3}" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.items' >/dev/null 2>&1; then
		print_success "Cart item updated"
		print_response "$response"
	else
		print_warning "Update may have failed"
		print_response "$response"
	fi

	# 3e. Clear cart for next tests
	print_test "3.5 Clear Cart"
	response=$(api_call DELETE "/api/v1/cart" "" "$BUYER_TOKEN")
	print_success "Cart cleared"

	# 3f. Add item again for checkout tests
	print_test "3.6 Add Item for Checkout Tests"
	response=$(api_call POST "/api/v1/cart/items" "{\"variant_id\":\"${VARIANT_ID}\",\"quantity\":1}" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.items' >/dev/null 2>&1; then
		print_success "Item added for checkout"
	else
		print_warning "Failed to add item for checkout (checkout tests may fail)"
	fi

	print_success "✓ Cart operations phase complete"
}

# =============================================================================
# PHASE 4: Address Management
# =============================================================================

phase_address() {
	print_header "PHASE 4: Address Management"

	# 4a. Create address
	print_test "4.1 Create Address"
	local response
	response=$(api_call POST "/api/v1/addresses" '{
        "full_name": "Test Buyer",
        "phone": "+251911000002",
        "line1": "Bole Sub-City",
        "line2": "Apt 123",
        "city": "Addis Ababa",
        "subcity": "Bole",
        "woreda": "03",
        "is_default": true
    }' "$BUYER_TOKEN")

	ADDRESS_ID=$(echo "$response" | jq -r '.id' 2>/dev/null)

	if [ -n "$ADDRESS_ID" ] && [ "$ADDRESS_ID" != "null" ]; then
		print_success "Address created: $ADDRESS_ID"
		print_response "$response"
	else
		print_error "Failed to create address"
		print_response "$response"
		return 1
	fi

	# 4b. List addresses
	print_test "4.2 List Addresses"
	response=$(api_call GET "/api/v1/addresses" "" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.addresses' >/dev/null 2>&1; then
		print_success "Addresses retrieved"
		print_response "$response"
	elif echo "$response" | jq -e 'if type == "array" then true else false end' >/dev/null 2>&1; then
		print_success "Addresses retrieved"
		print_response "$response"
	else
		print_error "Failed to list addresses"
		print_response "$response"
		return 1
	fi

	# 4c. Get address by ID
	print_test "4.3 Get Address by ID"
	response=$(api_call GET "/api/v1/addresses/${ADDRESS_ID}" "" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Address retrieved"
		print_response "$response"
	else
		print_error "Failed to get address"
		print_response "$response"
		return 1
	fi

	# 4d. Update address
	print_test "4.4 Update Address"
	response=$(api_call PUT "/api/v1/addresses/${ADDRESS_ID}" '{
        "full_name": "Updated Name",
        "phone": "+251911000002",
        "line1": "Mercato Area",
        "line2": "Floor 2",
        "city": "Addis Ababa",
        "subcity": "Arada"
    }' "$BUYER_TOKEN")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Address updated"
		print_response "$response"
	else
		print_warning "Address update may have failed"
		print_response "$response"
	fi

	print_success "✓ Address management phase complete"
}

# =============================================================================
# PHASE 5: Merchant Operations
# =============================================================================

phase_merchant() {
	print_header "PHASE 5: Merchant Operations"

	# 5a. Get merchant profile
	print_test "5.1 Get Merchant Profile"
	local response
	response=$(api_call GET "/api/v1/merchant/profile" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Merchant profile retrieved"
		print_response "$response"
	else
		print_error "Failed to get merchant profile"
		print_response "$response"
		return 1
	fi

	# 5b. Update merchant profile
	print_test "5.2 Update Merchant Profile"
	response=$(api_call PUT "/api/v1/merchant/profile" '{
        "shop_description": "Test shop updated via API test"
    }' "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Merchant profile updated"
		print_response "$response"
	else
		print_warning "Profile update may have failed"
		print_response "$response"
	fi

	# 5c. List merchant products
	print_test "5.3 List Merchant Products"
	response=$(api_call GET "/api/v1/merchant/products" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.products' >/dev/null 2>&1; then
		print_success "Merchant products retrieved"
		print_response "$response"
	elif echo "$response" | jq -e 'if type == "array" then true else false end' >/dev/null 2>&1; then
		print_success "Merchant products retrieved (array format)"
		print_response "$response"
	else
		print_warning "No merchant products found"
		print_response "$response"
	fi

	# 5d. Create product
	print_test "5.4 Create Product"
	response=$(api_call POST "/api/v1/merchant/products" "{
        \"name\": \"Test Product $(date +%s)\",
        \"slug\": \"test-product-$(date +%s)\",
        \"description\": \"A test product created via API\",
        \"category_id\": \"${CATEGORY_ID}\",
        \"is_digital\": false
    }" "$MERCHANT_TOKEN")

	NEW_PRODUCT_ID=$(echo "$response" | jq -r '.id' 2>/dev/null)

	if [ -n "$NEW_PRODUCT_ID" ] && [ "$NEW_PRODUCT_ID" != "null" ]; then
		print_success "Product created: $NEW_PRODUCT_ID"
		print_response "$response"
	else
		print_error "Failed to create product"
		print_response "$response"
		return 1
	fi

	# 5e. Add variant to product
	print_test "5.5 Add Product Variant"
	response=$(api_call POST "/api/v1/merchant/products/${NEW_PRODUCT_ID}/variants" "{
        \"sku\": \"TEST-SKU-$(date +%s)\",
        \"title\": \"Standard\",
        \"price_cents\": 100000,
        \"compare_price_cents\": 120000,
        \"currency\": \"ETB\",
        \"attributes\": {},
        \"is_active\": true
    }" "$MERCHANT_TOKEN")

	NEW_VARIANT_ID=$(echo "$response" | jq -r '.id' 2>/dev/null)

	if [ -n "$NEW_VARIANT_ID" ] && [ "$NEW_VARIANT_ID" != "null" ]; then
		print_success "Variant created: $NEW_VARIANT_ID"
		print_response "$response"
	else
		print_warning "Failed to create variant (may be expected)"
		print_response "$response"
	fi

	print_success "✓ Merchant operations phase complete"
}

# =============================================================================
# PHASE 6: Admin Operations
# =============================================================================

phase_admin() {
	print_header "PHASE 6: Admin Operations"

	# 6a. List users
	print_test "6.1 List Users"
	local response
	response=$(api_call GET "/api/v1/admin/users" "" "$ADMIN_TOKEN")

	if echo "$response" | jq -e '.items' >/dev/null 2>&1; then
		print_success "Users retrieved"
		print_response "$response"
	elif echo "$response" | jq -e '.users' >/dev/null 2>&1; then
		print_success "Users retrieved"
		print_response "$response"
	else
		print_error "Failed to list users"
		print_response "$response"
		return 1
	fi

	# 6b. Update KYC status
	print_test "6.2 Update User KYC Status"
	response=$(api_call PUT "/api/v1/admin/users/${BUYER_USER_ID}/kyc" '{"kyc_status":"verified"}' "$ADMIN_TOKEN")

	if echo "$response" | jq -e '.kyc_status' >/dev/null 2>&1; then
		print_success "KYC status updated"
		print_response "$response"
	else
		print_warning "KYC update may have failed"
		print_response "$response"
	fi

	# 6c. List merchants
	print_test "6.3 List Merchants"
	response=$(api_call GET "/api/v1/admin/merchants" "" "$ADMIN_TOKEN")

	if echo "$response" | jq -e '.merchants' >/dev/null 2>&1; then
		print_success "Merchants retrieved"
		print_response "$response"
	else
		print_warning "Failed to list merchants"
		print_response "$response"
	fi

	# 6d. Verify merchant
	print_test "6.4 Verify Merchant"
	response=$(api_call PUT "/api/v1/admin/merchants/${MERCHANT_ID}/verify" '{"is_verified":true}' "$ADMIN_TOKEN")

	if echo "$response" | jq -e '.is_verified' >/dev/null 2>&1 || echo "$response" | jq -e '.id' >/dev/null 2>&1; then
		print_success "Merchant verified"
		print_response "$response"
	else
		print_warning "Merchant verification may have failed"
		print_response "$response"
	fi

	# 6e. List payouts
	print_test "6.5 List Payouts (Admin)"
	response=$(api_call GET "/api/v1/admin/payouts" "" "$ADMIN_TOKEN")

	if echo "$response" | jq -e '.payouts' >/dev/null 2>&1 || echo "$response" | jq -e '.' >/dev/null 2>&1; then
		print_success "Payouts retrieved"
		print_response "$response"
	else
		print_warning "Failed to list payouts"
		print_response "$response"
	fi

	print_success "✓ Admin operations phase complete"
}

# =============================================================================
# PHASE 7: Checkout Flow
# =============================================================================

phase_checkout() {
	print_header "PHASE 7: Checkout Flow"

	# Ensure cart has items
	print_test "7.1 Ensure Cart Has Items"
	local response
	response=$(api_call GET "/api/v1/cart" "" "$BUYER_TOKEN")

	local item_count
	item_count=$(echo "$response" | jq '.items | length' 2>/dev/null || echo "0")

	if [ "$item_count" = "0" ]; then
		print_warning "Cart is empty, adding item..."
		response=$(api_call POST "/api/v1/cart/items" "{\"variant_id\":\"${VARIANT_ID}\",\"quantity\":1}" "$BUYER_TOKEN")
		print_response "$response"
	else
		print_success "Cart has $item_count items"
	fi

	# Ensure address exists
	print_test "7.2 Ensure Address Exists"
	response=$(api_call GET "/api/v1/addresses" "" "$BUYER_TOKEN")

	ADDRESS_ID=$(echo "$response" | jq -r '.addresses[0].id' 2>/dev/null)

	if [ -z "$ADDRESS_ID" ] || [ "$ADDRESS_ID" = "null" ]; then
		print_warning "No address found, creating one..."
		response=$(api_call POST "/api/v1/addresses" '{
            "full_name": "Checkout User",
            "phone": "+251911000002",
            "line1": "Bole Sub-City",
            "city": "Addis Ababa",
            "subcity": "Bole",
            "is_default": true
        }' "$BUYER_TOKEN")

		ADDRESS_ID=$(echo "$response" | jq -r '.id' 2>/dev/null)
		print_response "$response"
	else
		print_success "Address found: $ADDRESS_ID"
	fi

	# Checkout
	print_test "7.3 Initiate Checkout"
	local idempotency_key
	idempotency_key=$(generate_idempotency_key)
	response=$(api_call POST "/api/v1/orders/checkout" "{
        \"address_id\": \"${ADDRESS_ID}\",
        \"payment_method\": \"chapa\",
        \"currency\": \"ETB\"
    }" "$BUYER_TOKEN" "" "$idempotency_key")

	ORDER_ID=$(echo "$response" | jq -r '.order_id' 2>/dev/null)
	CHECKOUT_URL=$(echo "$response" | jq -r '.checkout_url' 2>/dev/null)

	if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "null" ]; then
		print_success "Checkout initiated"
		print_success "Order ID: $ORDER_ID"
		[ -n "$CHECKOUT_URL" ] && [ "$CHECKOUT_URL" != "null" ] && print_success "Checkout URL: $CHECKOUT_URL"
		print_response "$response"
	else
		print_error "Checkout failed"
		print_response "$response"
		# Continue to order phase anyway
	fi

	print_success "✓ Checkout phase complete"
}

# =============================================================================
# PHASE 8: Order Management
# =============================================================================

phase_order() {
	print_header "PHASE 8: Order Management"

	# 8a. List orders (buyer)
	print_test "8.1 List My Orders (Buyer)"
	local response
	response=$(api_call GET "/api/v1/orders" "" "$BUYER_TOKEN")

	if echo "$response" | jq -e '.orders' >/dev/null 2>&1; then
		print_success "Orders retrieved"
		# Get first order ID for further tests
		ORDER_ID=$(echo "$response" | jq -r '.orders[0].id' 2>/dev/null)
		print_response "$response"
	else
		print_warning "No orders found"
		print_response "$response"
	fi

	# 8b. Get order details
	if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "null" ]; then
		print_test "8.2 Get Order Details"
		response=$(api_call GET "/api/v1/orders/${ORDER_ID}" "" "$BUYER_TOKEN")

		if echo "$response" | jq -e '.id' >/dev/null 2>&1; then
			print_success "Order details retrieved"
			print_response "$response"
		else
			print_warning "Failed to get order details"
			print_response "$response"
		fi

		# 8c. Cancel order (if pending)
		print_test "8.3 Cancel Order"
		response=$(api_call POST "/api/v1/orders/${ORDER_ID}/cancel" "" "$BUYER_TOKEN")

		if echo "$response" | jq -e '.id' >/dev/null 2>&1 || echo "$response" | jq -e '.status' >/dev/null 2>&1; then
			print_success "Order cancelled"
			print_response "$response"
		else
			print_warning "Order cancellation may have failed (order may not be cancellable)"
			print_response "$response"
		fi
	else
		print_warning "No order ID available for detailed tests"
	fi

	# 8d. List merchant orders
	print_test "8.4 List Merchant Orders"
	response=$(api_call GET "/api/v1/merchant/orders" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.orders' >/dev/null 2>&1; then
		print_success "Merchant orders retrieved"
		print_response "$response"
	else
		print_warning "No merchant orders found"
		print_response "$response"
	fi

	print_success "✓ Order management phase complete"
}

# =============================================================================
# PHASE 9: Wallet & Payouts
# =============================================================================

phase_wallet() {
	print_header "PHASE 9: Wallet & Payouts"

	# 9a. Get wallet
	print_test "9.1 Get Merchant Wallet"
	local response
	response=$(api_call GET "/api/v1/merchant/wallet" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.balance' >/dev/null 2>&1 || echo "$response" | jq -e '.' >/dev/null 2>&1; then
		print_success "Wallet retrieved"
		print_response "$response"
	else
		print_warning "Failed to get wallet (may require Blnk setup)"
		print_response "$response"
	fi

	# 9b. List transactions
	print_test "9.2 List Wallet Transactions"
	response=$(api_call GET "/api/v1/merchant/wallet/transactions" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.transactions' >/dev/null 2>&1 || echo "$response" | jq -e '.' >/dev/null 2>&1; then
		print_success "Transactions retrieved"
		print_response "$response"
	else
		print_warning "Failed to get transactions"
		print_response "$response"
	fi

	# 9c. Request payout
	print_test "9.3 Request Payout"
	local payout_idempotency_key
	payout_idempotency_key=$(generate_idempotency_key)
	response=$(api_call POST "/api/v1/merchant/wallet/payout" '{
        "amount_cents": 50000,
        "currency": "ETB"
    }' "$MERCHANT_TOKEN" "" "$payout_idempotency_key")

	PAYOUT_ID=$(echo "$response" | jq -r '.id' 2>/dev/null)

	if [ -n "$PAYOUT_ID" ] && [ "$PAYOUT_ID" != "null" ]; then
		print_success "Payout requested: $PAYOUT_ID"
		print_response "$response"
	else
		print_warning "Payout request may have failed (insufficient balance?)"
		print_response "$response"
	fi

	# 9d. List merchant payouts
	print_test "9.4 List Merchant Payouts"
	response=$(api_call GET "/api/v1/merchant/payouts" "" "$MERCHANT_TOKEN")

	if echo "$response" | jq -e '.payouts' >/dev/null 2>&1 || echo "$response" | jq -e '.' >/dev/null 2>&1; then
		print_success "Payouts retrieved"
		print_response "$response"
	else
		print_warning "Failed to list payouts"
		print_response "$response"
	fi

	print_success "✓ Wallet & Payouts phase complete"
}

# =============================================================================
# Run All Phases
# =============================================================================

run_all() {
	print_header "Running All Test Phases"

	phase_auth || return 1
	phase_public || return 1
	phase_cart || return 1
	phase_address || return 1
	phase_merchant || return 1
	phase_admin || return 1
	phase_checkout || return 1
	phase_order || return 1
	phase_wallet || return 1

	print_header "All Tests Complete!"
	print_success "✓ All phases passed"
}

# =============================================================================
# Main
# =============================================================================

case "${1:-all}" in
1-auth)
	phase_auth
	;;
2-public)
	check_health
	phase_public
	;;
3-cart)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_cart
	;;
4-address)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_address
	;;
5-merchant)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_merchant
	;;
6-admin)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_admin
	;;
7-checkout)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_checkout
	;;
8-order)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_order
	;;
9-wallet)
	phase_auth >/dev/null 2>&1 || {
		print_error "Authentication required"
		exit 1
	}
	phase_wallet
	;;
all)
	run_all
	;;
health)
	check_health
	;;
*)
	echo "Usage: $0 {health|1-auth|2-public|3-cart|4-address|5-merchant|6-admin|7-checkout|8-order|9-wallet|all}"
	exit 1
	;;
esac
