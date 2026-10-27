#!/usr/bin/env bash
# Calls every API route against a running API + Blnk, with payments going through the
# mock Chapa's hosted checkout (Telebirr, CBE Birr, card), and checks the data each one
# returns. At the end it compares the routes it called with `go run ./cmd/routes` and
# fails if any route never returned a 2xx.
#
# Requires: API running with ENVIRONMENT=development, CHAPA_MODE=mock, SMS_PROVIDER=log,
# OTP_DEV_CODE set; `make local` services; jq, curl, openssl, go. Payout steps are quick
# with PAYOUT_RECHECK_INTERVAL=10s (the .env.example default).
# Usage: ./scripts/e2e-endpoints.sh [base_url]
set -euo pipefail

BASE="${1:-http://localhost:8080}"
API="$BASE/api/v1"
MOCK="${CHAPA_MOCK_URL:-$BASE/mock-chapa}"
WEBHOOK_SECRET="${CHAPA_WEBHOOK_SECRET:-dev-webhook-secret}"
OTP_CODE="${OTP_DEV_CODE:-123456}"
PG_CONTAINER="${PG_CONTAINER:-sukgebeya-postgres-dev}"
RUN="$(date +%s)$RANDOM"
SUFFIX=$(printf '%07d' $((RANDOM * RANDOM % 10000000)))
PASSWORD='Password123!'

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
CALLS="$WORK/calls"
HEADERS="$WORK/headers"
: >"$CALLS"

pass() { printf '  \033[32m✓\033[0m %s\n' "$1"; }
fail() { printf '  \033[31m✗\033[0m %s\n' "$1"; exit 1; }
step() { printf '\n\033[1m%s\033[0m\n' "$1"; }

# req METHOD PATH [token] [body] [extra header]: calls the API and records the route.
# Paths starting with / outside /api/v1 (health, swagger, the legacy webhook) are given
# with a leading "^".
req() {
	local method=$1 path=$2 token=${3:-} body=${4:-} extra=${5:-} url
	if [[ $path == ^* ]]; then url="$BASE${path#^}"; else url="$API$path"; fi
	local args=(-s -D "$HEADERS" -w '\n%{http_code}' -X "$method" "$url" -H 'Content-Type: application/json')
	[ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
	[ -n "$extra" ] && args+=(-H "$extra")
	[ -n "$body" ] && args+=(-d "$body")
	local out
	out=$(curl "${args[@]}")
	echo "$method ${url#"$BASE"} ${out##*$'\n'}" | sed 's/?[^ ]*//' >>"$CALLS"
	printf '%s' "$out"
}
expect() {
	local want=$1 out=$2 label=$3
	local code=${out##*$'\n'}
	BODY=${out%$'\n'*}
	[ "$code" = "$want" ] || { echo "$BODY" | jq . 2>/dev/null || echo "$BODY"; fail "$label (HTTP $code, want $want)"; }
	pass "$label"
}
# check <label> <jq filter that must be true on $BODY>
check() {
	[ "$(echo "$BODY" | jq -r "$2")" = "true" ] || { echo "$BODY" | jq . ; fail "$1"; }
	pass "$1"
}
header() { grep -i "^$1:" "$HEADERS" | head -1 | cut -d: -f2- | tr -d ' \r'; }
field() { req GET "$1" "$2" | sed '$d' | jq -r "$3"; }
wait_for() {
	local label=$1 cmd=$2 want=$3 timeout=${4:-30} got=""
	for _ in $(seq 1 $((timeout * 2))); do
		got=$(eval "$cmd")
		[ "$got" = "$want" ] && { pass "$label"; return; }
		sleep 0.5
	done
	fail "$label (got $got, want $want)"
}
register() {
	local name=$1 phone=$2 out
	out=$(req POST /register "" "{\"phone\":\"$phone\",\"email\":\"$name-$RUN@test.com\",\"password\":\"$PASSWORD\"}")
	expect 201 "$out" "register $name"
	printf -v "TOKEN_$name" '%s' "$(echo "$BODY" | jq -r .token.access_token)"
	printf -v "REFRESH_$name" '%s' "$(echo "$BODY" | jq -r .token.refresh_token)"
	printf -v "ID_$name" '%s' "$(echo "$BODY" | jq -r .user.id)"
}
verify_phone() {
	local out
	out=$(req POST /auth/otp/send "$1")
	expect 202 "$out" "$2: SMS code sent"
	out=$(req POST /auth/otp/verify "$1" "{\"code\":\"$OTP_CODE\"}")
	expect 200 "$out" "$2: phone verified"
}

# checkout_order <qty> <label>: fills the buyer's cart and starts a payment.
checkout_order() {
	local out
	out=$(req POST /cart/items "$BUYER" "{\"variant_id\":\"$VARIANT\",\"quantity\":$1}")
	expect 200 "$out" "$2: add to cart"
	out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\",\"notes\":\"Call before you come\"}" "Idempotency-Key: $2-$RUN")
	expect 201 "$out" "$2: checkout"
	ORDER=$(echo "$BODY" | jq -r .order_id)
	TXREF=$(echo "$BODY" | jq -r .chapa_tx_ref)
	PAYMENT_URL=$(echo "$BODY" | jq -r .payment_url)
}
# chapa <action> <method> [phone]: submits the mock's hosted checkout like the buyer would.
chapa() {
	local code location
	code=$(curl -s -o /dev/null -D "$HEADERS" -w '%{http_code}' -X POST "$PAYMENT_URL" \
		--data-urlencode "action=$1" --data-urlencode "method=$2" --data-urlencode "phone=${3:-}")
	location=$(header location)
	[ "$code" = "303" ] || fail "Chapa checkout: $1 with $2 (HTTP $code)"
	pass "Chapa checkout: $1 with $2, back to $location"
}
order_status() { field "/orders/$ORDER" "$BUYER" .order.status; }
mock_status() {
	curl -s -H 'Authorization: Bearer CHASECK_TEST-mock' "$MOCK/v1/transaction/verify/$TXREF" | jq -r .data.status
}

step "Health and docs"
out=$(req GET ^/health)
expect 200 "$out" "health"
check "database, ledger and redis are up" '.checks.database == "ok" and .checks.ledger == "ok" and .checks.redis == "ok"'
out=$(req GET ^/swagger/doc.json)
expect 200 "$out" "swagger spec"
out=$(curl -s -o /dev/null -w '%{http_code}' "$MOCK/")
[ "$out" = "200" ] || fail "mock Chapa is served at $MOCK (HTTP $out); is CHAPA_MODE=mock?"
pass "mock Chapa dashboard"

step "Accounts and sessions"
register buyer "09${SUFFIX}1"
register seller "09${SUFFIX}2"
register admin "09${SUFFIX}3"
register extra "09${SUFFIX}4"
BUYER=$TOKEN_buyer
SELLER=$TOKEN_seller

out=$(req POST /login "" "{\"identifier\":\"09${SUFFIX}1\",\"password\":\"$PASSWORD\"}")
expect 200 "$out" "log in with the phone number"
OLD_REFRESH=$(echo "$BODY" | jq -r .token.refresh_token)
out=$(req POST /refresh "" "{\"refresh_token\":\"$OLD_REFRESH\"}")
expect 200 "$out" "refresh rotates the tokens"
BUYER=$(echo "$BODY" | jq -r .access_token)
out=$(req POST /refresh "" "{\"refresh_token\":\"$OLD_REFRESH\"}")
expect 401 "$out" "a used refresh token is refused"

out=$(req GET /users/me "$BUYER")
expect 200 "$out" "read my profile"
check "profile has the local phone and email" ".email == \"buyer-$RUN@test.com\" and .phone_verified == false"
out=$(req PUT /users/me "$BUYER" '{"full_name":"Abebe Bikila"}')
expect 200 "$out" "update my name"
check "name saved" '.full_name == "Abebe Bikila"'
verify_phone "$BUYER" "buyer"
verify_phone "$SELLER" "seller"

out=$(req POST /auth/password/reset/request "" "{\"phone\":\"09${SUFFIX}4\"}")
expect 202 "$out" "request a password reset code"
out=$(req POST /auth/password/reset/confirm "" "{\"phone\":\"09${SUFFIX}4\",\"code\":\"$OTP_CODE\",\"new_password\":\"NewPassword456!\"}")
expect 200 "$out" "set a new password with the code"
out=$(req POST /login "" "{\"identifier\":\"extra-$RUN@test.com\",\"password\":\"NewPassword456!\"}")
expect 200 "$out" "log in with the new password"

docker exec "$PG_CONTAINER" psql -U postgres -d app -qc "UPDATE users SET role='admin' WHERE id='$ID_admin'" >/dev/null
out=$(req POST /login "" "{\"identifier\":\"admin-$RUN@test.com\",\"password\":\"$PASSWORD\"}")
expect 200 "$out" "admin logs in"
ADMIN=$(echo "$BODY" | jq -r .token.access_token)

step "Categories"
out=$(req POST /admin/categories "$ADMIN" "{\"name\":\"Clothing $RUN\",\"slug\":\"clothing-$RUN\",\"is_active\":true}")
expect 201 "$out" "admin creates a category"
CATEGORY=$(echo "$BODY" | jq -r .id)
out=$(req POST /admin/categories "$ADMIN" "{\"name\":\"Old $RUN\",\"slug\":\"old-$RUN\"}")
expect 201 "$out" "admin creates a second category"
OLD_CATEGORY=$(echo "$BODY" | jq -r .id)
out=$(req PUT "/admin/categories/$CATEGORY" "$ADMIN" "{\"name\":\"Traditional clothing $RUN\",\"sort_order\":1}")
expect 200 "$out" "admin renames it"
out=$(req GET /categories)
expect 200 "$out" "list categories"
check "the category is listed" "[.[] | select(.id == \"$CATEGORY\")] | length == 1"
out=$(req GET "/categories/$CATEGORY")
expect 200 "$out" "category by id"
check "category has the new name" ".name == \"Traditional clothing $RUN\""
out=$(req GET "/categories/slug/clothing-$RUN")
expect 200 "$out" "category by slug"
out=$(req DELETE "/admin/categories/$OLD_CATEGORY" "$ADMIN")
expect 204 "$out" "admin deletes the unused category"

step "Shop"
out=$(req POST /merchant/onboard "$SELLER" "{\"shop_name\":\"Tibeb Shop $RUN\",\"shop_slug\":\"tibeb-$RUN\",\"contact_phone\":\"09${SUFFIX}2\",\"city\":\"Addis Ababa\",\"delivery_fee_cents\":15000}")
expect 201 "$out" "seller opens a shop with a 150 ETB delivery fee"
MERCHANT_ID=$(echo "$BODY" | jq -r .merchant_id)
WALLET=$(echo "$BODY" | jq -r .blnk_balance_id)
out=$(req GET /merchant/profile "$SELLER")
expect 200 "$out" "seller reads the shop profile"
check "shop has its slug, fee and wallet" ".shop_slug == \"tibeb-$RUN\" and .delivery_fee_cents == 15000 and .has_wallet == true"
out=$(req PUT /merchant/profile "$SELLER" '{"shop_description":"Handwoven kemis from Gondar","free_delivery_threshold_cents":1000000}')
expect 200 "$out" "seller sets a description and free delivery over 10,000 ETB"
check "free delivery threshold saved" '.free_delivery_threshold_cents == 1000000'
out=$(req PUT /merchant/profile "$SELLER" '{"clear_free_delivery_threshold":true}')
expect 200 "$out" "seller removes free delivery"
check "threshold cleared, description kept" '.free_delivery_threshold_cents == null and .shop_description == "Handwoven kemis from Gondar"'
out=$(req GET "/shops/tibeb-$RUN")
expect 200 "$out" "public shop page"
check "shop page shows the name and fee" ".shop_name == \"Tibeb Shop $RUN\" and .delivery_fee_cents == 15000"

step "Catalog"
out=$(req POST /merchant/products "$SELLER" "{\"name\":\"Habesha Kemis $RUN\",\"status\":\"active\",\"category_id\":\"$CATEGORY\"}")
expect 201 "$out" "create a product"
PRODUCT=$(echo "$BODY" | jq -r .id)
SLUG=$(echo "$BODY" | jq -r .slug)
out=$(req PUT "/merchant/products/$PRODUCT" "$SELLER" '{"description":"Cotton, hand embroidered"}')
expect 200 "$out" "add a description"
out=$(req POST "/merchant/products/$PRODUCT/variants" "$SELLER" "{\"sku\":\"KEMIS-M-$RUN\",\"title\":\"Size M\",\"price_cents\":250000,\"currency\":\"ETB\",\"is_active\":true,\"stock_quantity\":5}")
expect 201 "$out" "add option Size M"
VARIANT=$(echo "$BODY" | jq -r .id)
out=$(req POST "/merchant/products/$PRODUCT/variants" "$SELLER" "{\"sku\":\"KEMIS-L-$RUN\",\"title\":\"Size L\",\"price_cents\":260000,\"currency\":\"ETB\",\"stock_quantity\":2}")
expect 201 "$out" "add option Size L"
VARIANT_L=$(echo "$BODY" | jq -r .id)
out=$(req PUT "/merchant/products/$PRODUCT/variants/$VARIANT" "$SELLER" '{"compare_price_cents":300000}')
expect 200 "$out" "set a compare-at price on Size M"
out=$(req PUT "/merchant/products/$PRODUCT/variants/$VARIANT/stock" "$SELLER" '{"quantity":20}')
expect 200 "$out" "set Size M stock to 20"
out=$(req DELETE "/merchant/products/$PRODUCT/variants/$VARIANT_L" "$SELLER")
expect 204 "$out" "remove Size L"
out=$(req POST "/merchant/products/$PRODUCT/images" "$SELLER" '{"url":"https://cdn.example.com/kemis.jpg","is_primary":true}')
expect 201 "$out" "add the main image"
out=$(req POST "/merchant/products/$PRODUCT/images" "$SELLER" '{"url":"https://cdn.example.com/kemis-back.jpg"}')
expect 201 "$out" "add a second image"
IMAGE=$(echo "$BODY" | jq -r .id)
out=$(req DELETE "/merchant/products/$PRODUCT/images/$IMAGE" "$SELLER")
expect 204 "$out" "remove the second image"

out=$(req GET "/merchant/products/$PRODUCT" "$SELLER")
expect 200 "$out" "seller reads the product"
check "one option, still named Size M, at 2,500 ETB with 20 in stock, compare-at 3,000, one image" \
	'(.variants | length) == 1 and .variants[0].title == "Size M" and .variants[0].price_cents == 250000 and .variants[0].compare_price_cents == 300000 and .variants[0].available_quantity == 20 and (.images | length) == 1'
out=$(req GET "/merchant/products?status=active" "$SELLER")
expect 200 "$out" "seller lists active products"
check "the product is listed" "[.items[] | select(.id == \"$PRODUCT\")] | length == 1"
out=$(req GET "/products?merchant_id=$MERCHANT_ID")
expect 200 "$out" "public list of the shop's products"
check "listing shows price, stock and image" '.items[0].min_price_cents == 250000 and .items[0].in_stock == true and .items[0].primary_image_url == "https://cdn.example.com/kemis.jpg"'
out=$(req GET "/products/$SLUG")
expect 200 "$out" "public product page"
check "product page has the description, option and shop" ".product.description == \"Cotton, hand embroidered\" and (.variants | length) == 1 and .shop.delivery_fee_cents == 15000"

out=$(req POST /merchant/products "$SELLER" "{\"name\":\"Netela $RUN\",\"status\":\"draft\"}")
expect 201 "$out" "create a draft product"
DRAFT=$(echo "$BODY" | jq -r .id)
DRAFT_SLUG=$(echo "$BODY" | jq -r .slug)
out=$(req GET "/products/$DRAFT_SLUG")
expect 404 "$out" "buyers can't open a draft"
out=$(req DELETE "/merchant/products/$DRAFT" "$SELLER")
expect 204 "$out" "delete the draft"

step "Guest cart, then sign in"
out=$(req POST /cart/items "" "{\"variant_id\":\"$VARIANT\",\"quantity\":1}")
expect 200 "$out" "guest adds to cart"
check "the new line names the product and option" ".product_name == \"Habesha Kemis $RUN\" and .variant_title == \"Size M\""
SESSION=$(header x-session-id)
[ -n "$SESSION" ] || fail "guest cart returns a session ID"
out=$(req PUT "/cart/items/$VARIANT" "" '{"quantity":3}' "X-Session-ID: $SESSION")
expect 200 "$out" "guest changes the quantity to 3"
check "the line is 3 × Size M = 7,500 ETB" ".quantity == 3 and .subtotal_cents == 750000 and .variant_title == \"Size M\" and .product_name == \"Habesha Kemis $RUN\""
out=$(req DELETE "/cart/items/$VARIANT" "" "" "X-Session-ID: $SESSION")
expect 204 "$out" "guest removes the line"
out=$(req POST /cart/items "" "{\"variant_id\":\"$VARIANT\",\"quantity\":2}" "X-Session-ID: $SESSION")
expect 200 "$out" "guest adds 2 again"
out=$(req DELETE /cart "" "" "X-Session-ID: $SESSION")
expect 204 "$out" "guest empties the cart"
out=$(req POST /cart/items "" "{\"variant_id\":\"$VARIANT\",\"quantity\":1}" "X-Session-ID: $SESSION")
expect 200 "$out" "guest adds 1"
out=$(req GET /cart "" "" "X-Session-ID: $SESSION")
expect 200 "$out" "guest reads the cart"
check "one Size M in the guest cart" ".item_count == 1 and .items[0].variant_title == \"Size M\" and .items[0].product_name == \"Habesha Kemis $RUN\""
out=$(req POST /cart/merge "$BUYER" "" "X-Session-ID: $SESSION")
expect 200 "$out" "signing in merges the guest cart"
out=$(req GET /cart "$BUYER")
expect 200 "$out" "buyer reads the cart"
check "the guest's item is in the buyer's cart" '.item_count == 1 and .total_cents == 250000'

step "Addresses"
out=$(req POST /addresses "$BUYER" "{\"full_name\":\"Abebe Bikila\",\"phone\":\"09${SUFFIX}1\",\"line1\":\"Bole, near Edna Mall, house 12\",\"city\":\"Addis Ababa\",\"subcity\":\"Bole\",\"woreda\":\"03\",\"is_default\":true}")
expect 201 "$out" "add a home address"
HOME_ADDRESS=$(echo "$BODY" | jq -r .id)
out=$(req POST /addresses "$BUYER" "{\"full_name\":\"Abebe Bikila\",\"phone\":\"09${SUFFIX}1\",\"line1\":\"Piassa, office 4\",\"city\":\"Addis Ababa\",\"subcity\":\"Arada\",\"woreda\":\"01\"}")
expect 201 "$out" "add an office address"
ADDRESS=$(echo "$BODY" | jq -r .id)
out=$(req PUT "/addresses/$ADDRESS" "$BUYER" '{"line2":"Second floor"}')
expect 200 "$out" "add details to the office address"
out=$(req PUT "/addresses/$ADDRESS/default" "$BUYER")
expect 200 "$out" "make the office the default"
out=$(req GET "/addresses/$ADDRESS" "$BUYER")
expect 200 "$out" "read the office address"
check "office is the default and has the details" '.is_default == true and .line2 == "Second floor"'
out=$(req DELETE "/addresses/$HOME_ADDRESS" "$BUYER")
expect 204 "$out" "delete the home address"
out=$(req GET /addresses "$BUYER")
expect 200 "$out" "list addresses"
check "only the office is left" "length == 1 and .[0].id == \"$ADDRESS\""

step "Paid with Telebirr, shipped, delivered"
out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\",\"notes\":\"Call before you come\"}" "Idempotency-Key: first-$RUN")
expect 201 "$out" "checkout the merged cart"
ORDER=$(echo "$BODY" | jq -r .order_id)
TXREF=$(echo "$BODY" | jq -r .chapa_tx_ref)
PAYMENT_URL=$(echo "$BODY" | jq -r .payment_url)
check "total is 2,500 + 150 delivery" '.total_cents == 265000'
[[ $PAYMENT_URL == "$MOCK/checkout/"* ]] || fail "payment link goes to the mock Chapa ($PAYMENT_URL)"
pass "payment link goes to the mock Chapa"
out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\"}" "Idempotency-Key: first-$RUN")
expect 201 "$out" "a repeated checkout returns the same order"
[ "$(header x-idempotent-replayed)" = "true" ] && [ "$(echo "$BODY" | jq -r .order_id)" = "$ORDER" ] || fail "replayed checkout"
pass "replayed from the idempotency cache"

page=$(curl -s "$PAYMENT_URL")
[[ $page == *"Pay 2,650.00 ETB"* ]] || fail "hosted checkout shows the amount"
pass "hosted checkout shows 2,650.00 ETB"
[ "$(mock_status)" = "pending" ] || fail "payment is pending at Chapa before paying"
chapa pay telebirr "09${SUFFIX}1"
wait_for "the mock's signed webhook marks the order paid" order_status "paid"
out=$(req POST "/orders/$ORDER/payment/refresh" "$BUYER")
expect 200 "$out" "re-check the payment"
check "still paid" '.status == "paid"'
out=$(req GET "/payments/chapa/callback?trx_ref=$TXREF")
expect 200 "$out" "Chapa callback (the mock also calls it)"
payload="{\"event\":\"charge.success\",\"tx_ref\":\"$TXREF\",\"status\":\"success\",\"currency\":\"ETB\"}"
sig=$(printf '%s' "$payload" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex | sed 's/^.* //')
out=$(req POST /webhooks/chapa "" "$payload" "x-chapa-signature: $sig")
expect 200 "$out" "a repeated webhook is accepted"
out=$(req POST ^/webhooks/chapa "" "$payload" "x-chapa-signature: $sig")
expect 200 "$out" "the legacy webhook path still works"
out=$(req POST /webhooks/chapa "" "$payload" "x-chapa-signature: bad")
expect 401 "$out" "a webhook with a bad signature is refused"

out=$(req GET /orders "$BUYER")
expect 200 "$out" "buyer lists orders"
check "the order is listed as paid" "[.items[] | select(.id == \"$ORDER\" and .status == \"paid\")] | length == 1"
out=$(req GET "/orders/$ORDER" "$BUYER")
expect 200 "$out" "buyer reads the order"
check "order has the item, note and office address" '.items[0].variant_title == "Size M" and .order.notes == "Call before you come" and .shipping_address.subcity == "Arada"'
out=$(req GET "/merchant/orders?status=paid" "$SELLER")
expect 200 "$out" "seller lists paid orders"
check "the order is waiting for the seller" "[.items[] | select(.id == \"$ORDER\")] | length == 1"
out=$(req GET "/merchant/orders/$ORDER" "$SELLER")
expect 200 "$out" "seller reads the order"
check "seller sees the delivery address and fee" '.shipping_address.line1 == "Piassa, office 4" and .order.platform_fee_cents == 12500'
out=$(req PUT "/merchant/orders/$ORDER/status" "$SELLER" '{"status":"processing"}')
expect 200 "$out" "seller accepts the order"
out=$(req PUT "/merchant/orders/$ORDER/ship" "$SELLER" '{"carrier":"Ethio Post","tracking_number":"EP123456789ET"}')
expect 200 "$out" "seller ships it"
out=$(req GET "/orders/$ORDER" "$BUYER")
expect 200 "$out" "buyer sees the shipment"
check "carrier and tracking number" '.order.status == "shipped" and .shipment.carrier == "Ethio Post" and .shipment.tracking_number == "EP123456789ET"'
out=$(req POST "/orders/$ORDER/confirm-delivery" "$BUYER")
expect 200 "$out" "buyer confirms delivery"
DELIVERED=$ORDER

step "Paid by card, cancelled by the buyer, refunded"
checkout_order 1 card-cancel
chapa pay card
wait_for "order paid" order_status "paid"
out=$(req POST "/orders/$ORDER/cancel" "$BUYER")
expect 200 "$out" "buyer cancels the paid order"
wait_for "order refunded" order_status "refunded"
wait_for "Chapa refunded the payment" mock_status "refunded"

step "Paid with CBE Birr, cancelled by the seller, refunded"
checkout_order 1 seller-cancel
chapa pay cbebirr "09${SUFFIX}1"
wait_for "order paid" order_status "paid"
out=$(req PUT "/merchant/orders/$ORDER/status" "$SELLER" '{"status":"processing"}')
expect 200 "$out" "seller accepts the order"
out=$(req POST "/merchant/orders/$ORDER/cancel" "$SELLER")
expect 200 "$out" "seller cancels and refunds the buyer"
wait_for "order refunded" order_status "refunded"

step "Dispute released to the seller"
checkout_order 1 dispute
chapa pay telebirr "09${SUFFIX}1"
wait_for "order paid" order_status "paid"
out=$(req PUT "/merchant/orders/$ORDER/status" "$SELLER" '{"status":"processing"}')
expect 200 "$out" "seller accepts the order"
out=$(req PUT "/merchant/orders/$ORDER/ship" "$SELLER" '{"carrier":"Self delivery"}')
expect 200 "$out" "seller ships it"
out=$(req POST "/orders/$ORDER/dispute" "$BUYER" '{"reason":"The colour is not what the photo showed"}')
expect 200 "$out" "buyer opens a dispute"
check "dispute reason saved" '.status == "disputed" and .dispute_reason == "The colour is not what the photo showed"'
out=$(req POST "/admin/orders/$ORDER/resolve" "$ADMIN" '{"outcome":"release"}')
expect 200 "$out" "admin releases the money to the seller"
wait_for "order delivered" order_status "delivered"

step "Payments that don't go through"
checkout_order 1 declined
chapa decline cbebirr
wait_for "a declined payment cancels the order" order_status "cancelled"
checkout_order 1 abandoned
code=$(curl -s -o /dev/null -w '%{http_code}' "$PAYMENT_URL")
[ "$code" = "200" ] || fail "buyer opens the checkout"
out=$(req POST "/orders/$ORDER/payment/refresh" "$BUYER")
expect 200 "$out" "buyer comes back without paying"
check "order still waits for payment" '.status == "pending_payment"'
out=$(req POST "/orders/$ORDER/cancel" "$BUYER")
expect 200 "$out" "buyer cancels the unpaid order"
check "cancelled" '.status == "cancelled"'
out=$(req GET "/merchant/products/$PRODUCT" "$SELLER")
expect 200 "$out" "seller reads the stock"
check "stock is back to 18 (two delivered)" '.variants[0].available_quantity == 18'

step "Wallet"
wait_for "two sales credited: 2 × (2,500 − 125 + 150) = 5,050.00 ETB" \
	"field /merchant/wallet '$SELLER' .balance_cents" "505000" 60
out=$(req GET /merchant/dashboard "$SELLER")
expect 200 "$out" "seller dashboard"
check "dashboard: 5,050.00 ETB available, total sales counted" '.available_cents == 505000 and .total_sales_cents > 0'
out=$(req GET /merchant/wallet/transactions "$SELLER")
expect 200 "$out" "wallet transactions"
check "two sale credits of 2,525.00 ETB" '[.items[] | select(.type == "sale_credit" and .amount_cents == 252500)] | length == 2'
out=$(req GET /merchant/wallet/banks "$SELLER")
expect 200 "$out" "banks for payouts"
check "Telebirr is offered" '[.[] | select(.slug == "telebirr")] | length == 1'

step "Verification and payouts to Telebirr"
out=$(req GET /admin/users "$ADMIN")
expect 200 "$out" "admin lists users"
out=$(req PUT "/admin/users/$ID_seller/kyc" "$ADMIN" '{"kyc_status":"verified"}')
expect 200 "$out" "admin verifies the seller's ID"
out=$(req GET /admin/merchants "$ADMIN")
expect 200 "$out" "admin lists shops"
check "the shop is listed, not yet verified" "[.items[] | select(.id == \"$MERCHANT_ID\" and .is_verified == false)] | length == 1"
out=$(req PUT "/admin/merchants/$MERCHANT_ID/verify" "$ADMIN" '{"is_verified":true}')
expect 200 "$out" "admin verifies the shop"
out=$(req POST "/admin/merchants/$MERCHANT_ID/wallet" "$ADMIN")
expect 200 "$out" "provisioning an existing wallet is a no-op"
check "the shop keeps its wallet" '.has_wallet == true'

payout() {
	out=$(req POST /merchant/wallet/payout "$SELLER" "{\"amount_cents\":$1,\"method\":\"telebirr\",\"destination\":\"$2\",\"account_name\":\"Tigist Bekele\",\"bank_code\":\"855\"}" "Idempotency-Key: payout-$3-$RUN")
	expect 201 "$out" "request $(($1 / 100)).00 ETB to Telebirr $2"
	PAYOUT=$(echo "$BODY" | jq -r .id)
}
payout_status() { field "/merchant/wallet/payouts/$PAYOUT" "$SELLER" .status; }

payout 100000 "09${SUFFIX}5" ok
out=$(req GET "/admin/payouts?status=pending" "$ADMIN")
expect 200 "$out" "admin lists payouts waiting for review"
check "the payout is waiting" "[.items[] | select(.id == \"$PAYOUT\")] | length == 1"
out=$(req POST "/admin/payouts/$PAYOUT/process" "$ADMIN")
expect 202 "$out" "admin approves it"
wait_for "the Telebirr transfer settles and the payout completes" payout_status "completed" 180

payout 20000 "09${SUFFIX}5" rejected
out=$(req POST "/admin/payouts/$PAYOUT/reject" "$ADMIN" '{"reason":"Name does not match the ID"}')
expect 200 "$out" "admin rejects the next one"
check "rejected with the reason" '.status == "failed" and .failure_reason == "Name does not match the ID"'

payout 30000 "0911110000" bad-number
out=$(req POST "/admin/payouts/$PAYOUT/process" "$ADMIN")
expect 202 "$out" "admin approves a payout to a number with no Telebirr wallet"
wait_for "the transfer fails at the provider" payout_status "failed" 180

out=$(req GET /merchant/wallet/payouts "$SELLER")
expect 200 "$out" "seller lists payouts"
check "completed, rejected and failed are all listed" '[.items[].status] | sort == ["completed", "failed", "failed"]'
out=$(req GET "/merchant/wallet/payouts/$PAYOUT" "$SELLER")
expect 200 "$out" "seller reads the failed payout"
check "failure reason shown" '.failure_reason == "transfer failed at provider"'
out=$(req GET /merchant/wallet "$SELLER")
expect 200 "$out" "seller reads the wallet"
check "only the completed 1,000 ETB left: 4,050.00 ETB available, nothing held" '.balance_cents == 405000 and .available_cents == 405000 and .pending_payout_cents == 0'

step "Chargeback after settlement"
TXREF=$(docker exec "$PG_CONTAINER" psql -U postgres -d app -Atc "SELECT chapa_tx_ref FROM orders WHERE id='$DELIVERED'")
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$MOCK/payments/$TXREF/reverse")
[ "$code" = "303" ] || fail "reverse the first payment at Chapa (HTTP $code)"
pass "Chapa reverses the first payment"
wait_for "the merchant is debited the 2,525.00 ETB credit" "field /merchant/wallet '$SELLER' .balance_cents" "152500" 60
out=$(req GET /merchant/wallet/transactions "$SELLER")
expect 200 "$out" "wallet transactions"
check "a reversal debit is recorded" '[.items[] | select(.type == "reversal_debit" and .amount_cents == 252500)] | length == 1'

step "Reconciliation and admin"
out=$(req GET /admin/reconciliation "$ADMIN")
expect 200 "$out" "reconciliation report"
check "this shop's wallet matches its transactions" "[.mismatches[]? | select(.balance_id == \"$WALLET\")] | length == 0"
out=$(req GET /users "$ADMIN")
expect 200 "$out" "admin lists accounts"
out=$(req GET "/users/$ID_buyer" "$ADMIN")
expect 200 "$out" "admin reads an account"
check "it's the buyer" '.full_name == "Abebe Bikila"'
out=$(req DELETE "/users/$ID_extra" "$ADMIN")
expect 204 "$out" "admin deletes an account"
out=$(req GET /users "$BUYER")
expect 403 "$out" "buyers can't list accounts"

step "Signing out"
out=$(req POST /logout "" "{\"refresh_token\":\"$REFRESH_seller\"}")
expect 204 "$out" "seller logs out"
out=$(req POST /refresh "" "{\"refresh_token\":\"$REFRESH_seller\"}")
expect 401 "$out" "the logged-out refresh token is refused"
out=$(req POST /login "" "{\"identifier\":\"09${SUFFIX}1\",\"password\":\"$PASSWORD\"}")
expect 200 "$out" "buyer logs in on a second device"
SECOND=$(echo "$BODY" | jq -r .token.refresh_token)
out=$(req POST /logout-all "$BUYER")
expect 204 "$out" "buyer logs out everywhere"
out=$(req POST /refresh "" "{\"refresh_token\":\"$SECOND\"}")
expect 401 "$out" "the other device's session is gone"

step "Route coverage"
ROUTES="$WORK/routes"
(cd "$(dirname "$0")/.." && go run ./cmd/routes) >"$ROUTES"
missing=$(awk '
	NR == FNR {
		method[NR] = $1; pattern[NR] = $2
		re = $2; gsub(/\{[^}]+\}/, "[^/]+", re); gsub(/\*/, ".*", re)
		regex[NR] = "^" re "$"; params[NR] = gsub(/\{/, "{", $2); n = NR
		next
	}
	$3 ~ /^2/ {
		best = 0
		for (i = 1; i <= n; i++)
			if (method[i] == $1 && $2 ~ regex[i] && (best == 0 || params[i] < params[best])) best = i
		if (best) hit[best] = 1
	}
	END { for (i = 1; i <= n; i++) if (!hit[i]) print "    " method[i] " " pattern[i] }
' "$ROUTES" "$CALLS")
total=$(wc -l <"$ROUTES")
if [ -n "$missing" ]; then
	echo "$missing"
	fail "routes never returned a 2xx: $(echo "$missing" | wc -l) of $total"
fi
pass "all $total routes returned a 2xx"

printf '\n\033[32mEvery endpoint works.\033[0m\n'
