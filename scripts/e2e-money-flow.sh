#!/usr/bin/env bash
# End-to-end check of the Ethiopian marketplace money flow against a running API + Blnk:
# phone registration (any local format) -> SMS verification -> merchant onboarding with
# delivery fee -> KYC -> stock -> cart -> checkout -> signed webhook -> paid -> ship ->
# deliver -> settlement (merchant credit, platform fee, Chapa fee) -> dispute + refund ->
# payout -> reconciliation.
#
# Requires: API running with ENVIRONMENT=development, CHAPA_MODE=mock, SMS_PROVIDER=log,
# OTP_DEV_CODE set; `make local` services; jq, curl, openssl.
# Usage: ./scripts/e2e-money-flow.sh [base_url]
set -euo pipefail

BASE="${1:-http://localhost:8080}"
API="$BASE/api/v1"
WEBHOOK_SECRET="${CHAPA_WEBHOOK_SECRET:-dev-webhook-secret}"
OTP_CODE="${OTP_DEV_CODE:-123456}"
BLNK="${BLNK_BASE_URL:-http://localhost:5001}"
BLNK_KEY="${BLNK_SECRET_KEY:-dev-blnk-secret}"
PG_CONTAINER="${PG_CONTAINER:-sukgebeya-postgres-dev}"
RUN="$(date +%s)$RANDOM"
SUFFIX=$(printf '%07d' $((RANDOM * RANDOM % 10000000)))

REVENUE_ID="${BLNK_REVENUE_BALANCE_ID:-}"
FEES_ID="${BLNK_PROCESSING_FEE_BALANCE_ID:-}"
if [ -f .env ]; then
	[ -n "$REVENUE_ID" ] || REVENUE_ID=$(grep '^BLNK_REVENUE_BALANCE_ID=' .env | cut -d= -f2)
	[ -n "$FEES_ID" ] || FEES_ID=$(grep '^BLNK_PROCESSING_FEE_BALANCE_ID=' .env | cut -d= -f2)
fi

pass() { printf '  \033[32m✓\033[0m %s\n' "$1"; }
fail() { printf '  \033[31m✗\033[0m %s\n' "$1"; exit 1; }
step() { printf '\n\033[1m%s\033[0m\n' "$1"; }

req() {
	local method=$1 path=$2 token=${3:-} body=${4:-} extra=${5:-}
	local args=(-s -w '\n%{http_code}' -X "$method" "$API$path" -H 'Content-Type: application/json')
	[ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
	[ -n "$extra" ] && args+=(-H "$extra")
	[ -n "$body" ] && args+=(-d "$body")
	curl "${args[@]}"
}

expect() {
	local want=$1 out=$2 label=$3
	local code=${out##*$'\n'}
	BODY=${out%$'\n'*}
	[ "$code" = "$want" ] || { echo "$BODY" | jq . 2>/dev/null || echo "$BODY"; fail "$label (HTTP $code, want $want)"; }
	pass "$label"
}

blnk_balance() {
	curl -s -H "X-Blnk-Key: $BLNK_KEY" "$BLNK/balances/$1" | jq -r '.balance'
}

field() {
	local path=$1 token=$2 query=$3
	req GET "$path" "$token" | sed '$d' | jq -r "$query"
}

# wait_for <label> <command> <expected> [timeout seconds, default 20]
wait_for() {
	local label=$1 cmd=$2 want=$3 timeout=${4:-20} got=""
	for _ in $(seq 1 $((timeout * 2))); do
		got=$(eval "$cmd")
		[ "$got" = "$want" ] && { pass "$label"; return; }
		sleep 0.5
	done
	fail "$label (got $got, want $want)"
}

register() {
	local name=$1 phone=$2 out
	out=$(req POST /register "" "{\"phone\":\"$phone\",\"email\":\"$name-$RUN@test.com\",\"password\":\"Password123!\"}")
	expect 201 "$out" "register $name with $phone"
	printf -v "TOKEN_$name" '%s' "$(echo "$BODY" | jq -r .token.access_token)"
	printf -v "ID_$name" '%s' "$(echo "$BODY" | jq -r .user.id)"
}

verify_phone() {
	local token=$1 label=$2 out
	out=$(req POST /auth/otp/send "$token")
	expect 202 "$out" "$label: SMS code sent"
	out=$(req POST /auth/otp/verify "$token" "{\"code\":\"$OTP_CODE\"}")
	expect 200 "$out" "$label: phone verified"
	[ "$(echo "$BODY" | jq -r .phone_verified)" = "true" ] || fail "$label phone_verified flag"
}

checkout() {
	local qty=$1 key=$2 out
	out=$(req POST /cart/items "$BUYER" "{\"variant_id\":\"$VARIANT\",\"quantity\":$qty}")
	expect 200 "$out" "add $qty to cart"
	out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\"}" "Idempotency-Key: $key")
	expect 201 "$out" "checkout"
	ORDER=$(echo "$BODY" | jq -r .order_id)
	TXREF=$(echo "$BODY" | jq -r .chapa_tx_ref)
	TOTAL=$(echo "$BODY" | jq -r .total_cents)
}

pay() {
	local payload sig code
	payload="{\"event\":\"charge.success\",\"tx_ref\":\"$TXREF\",\"status\":\"success\",\"currency\":\"ETB\"}"
	sig=$(printf '%s' "$payload" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex | sed 's/^.* //')
	code=$(curl -s -o /dev/null -w '%{http_code}' -XPOST "$API/webhooks/chapa" -H 'Content-Type: application/json' -H "x-chapa-signature: $sig" -d "$payload")
	[ "$code" = "200" ] || fail "signed webhook accepted (got $code)"
	wait_for "order is paid" "field /orders/$ORDER '$BUYER' .order.status" "paid"
}

ship() {
	local out
	out=$(req PUT "/merchant/orders/$ORDER/status" "$SELLER" '{"status":"processing"}')
	expect 200 "$out" "merchant starts processing"
	out=$(req PUT "/merchant/orders/$ORDER/ship" "$SELLER" '{"carrier":"Ethio Post"}')
	expect 200 "$out" "merchant ships"
}

step "Phone-first accounts"
register buyer "09${SUFFIX}1"
register seller "+251 7${SUFFIX}2"
register admin "9${SUFFIX}3"
BUYER=$TOKEN_buyer
SELLER=$TOKEN_seller
out=$(req POST /register "" "{\"phone\":\"2519${SUFFIX}1\",\"password\":\"Password123!\"}")
expect 409 "$out" "same number in another format is a duplicate"
out=$(req POST /login "" "{\"identifier\":\"+2519${SUFFIX}1\",\"password\":\"Password123!\"}")
expect 200 "$out" "log in with the +251 form of a number registered as 09…"
docker exec "$PG_CONTAINER" psql -U postgres -d app -qc "UPDATE users SET role='admin' WHERE id='$ID_admin'" >/dev/null
out=$(req POST /login "" "{\"identifier\":\"admin-$RUN@test.com\",\"password\":\"Password123!\"}")
expect 200 "$out" "admin login"
ADMIN=$(echo "$BODY" | jq -r .token.access_token)

step "Merchant onboarding"
ONBOARD="{\"shop_name\":\"ጥበብ ሱቅ $RUN\",\"contact_phone\":\"07${SUFFIX}2\",\"city\":\"Addis Ababa\",\"delivery_fee_cents\":15000,\"tin\":\"0012345678\"}"
out=$(req POST /merchant/onboard "$SELLER" "$ONBOARD")
expect 403 "$out" "onboarding requires a verified phone"
verify_phone "$SELLER" "seller"
out=$(req POST /merchant/onboard "$SELLER" "$ONBOARD")
expect 201 "$out" "onboard Amharic-named shop with 150 ETB delivery fee"
WALLET=$(echo "$BODY" | jq -r .blnk_balance_id)
MERCHANT_ID=$(echo "$BODY" | jq -r .merchant_id)
[ "$(curl -s -o /dev/null -w '%{http_code}' -H "X-Blnk-Key: $BLNK_KEY" "$BLNK/balances/$WALLET")" = "200" ] || fail "wallet exists in Blnk"
pass "wallet $WALLET exists in Blnk"

step "Catalog and stock"
out=$(req POST /merchant/products "$SELLER" "{\"name\":\"የሀበሻ ቀሚስ $RUN\",\"status\":\"active\"}")
expect 201 "$out" "create product with an Amharic name (slug generated)"
PRODUCT=$(echo "$BODY" | jq -r .id)
out=$(req POST "/merchant/products/$PRODUCT/variants" "$SELLER" "{\"sku\":\"KEMIS-$RUN\",\"title\":\"Size M\",\"price_cents\":250000,\"currency\":\"ETB\",\"is_active\":true,\"stock_quantity\":10}")
expect 201 "$out" "variant: 2,500 ETB, 10 in stock"
VARIANT=$(echo "$BODY" | jq -r .id)
out=$(req POST "/merchant/products/$PRODUCT/images" "$SELLER" '{"url":"https://cdn.example.com/kemis.jpg"}')
expect 201 "$out" "attach product image"
out=$(curl -s -w '\n%{http_code}' "$API/products?q=$(printf 'ቀሚስ %s' "$RUN" | jq -sRr @uri)")
expect 200 "$out" "Amharic search"
[ "$(echo "$BODY" | jq -r '.items[0].min_price_cents')" = "250000" ] || fail "listing shows price"
[ "$(echo "$BODY" | jq -r '.items[0].in_stock')" = "true" ] || fail "listing shows stock"
[ "$(echo "$BODY" | jq -r '.items[0].primary_image_url')" = "https://cdn.example.com/kemis.jpg" ] || fail "listing shows image"
pass "listing shows price, stock and image"

step "Checkout"
out=$(req POST /addresses "$BUYER" "{\"full_name\":\"Abebe Bikila\",\"phone\":\"09${SUFFIX}1\",\"line1\":\"Bole Sub-city, Woreda 03\",\"city\":\"Addis Ababa\",\"subcity\":\"Bole\",\"woreda\":\"03\",\"is_default\":true}")
expect 201 "$out" "add Addis address with subcity and woreda"
ADDRESS=$(echo "$BODY" | jq -r .id)
out=$(req POST /cart/items "$BUYER" "{\"variant_id\":\"$VARIANT\",\"quantity\":2}")
expect 200 "$out" "add 2 to cart"
out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\"}" "Idempotency-Key: blocked-$RUN")
expect 403 "$out" "checkout requires a verified phone"
verify_phone "$BUYER" "buyer"
out=$(req POST /orders/checkout "$BUYER" "{\"shipping_address_id\":\"$ADDRESS\"}" "Idempotency-Key: first-$RUN")
expect 201 "$out" "checkout"
ORDER=$(echo "$BODY" | jq -r .order_id)
TXREF=$(echo "$BODY" | jq -r .chapa_tx_ref)
TOTAL=$(echo "$BODY" | jq -r .total_cents)
[ "$TOTAL" = "515000" ] || fail "total = 5,000 ETB goods + 150 ETB delivery (got $TOTAL)"
pass "total is 5,150.00 ETB including delivery"
[ ${#TXREF} -le 50 ] || fail "tx_ref fits Chapa (len ${#TXREF})"
pass "tx_ref is ${#TXREF} chars"

REV_BEFORE=$(blnk_balance "$REVENUE_ID")
FEES_BEFORE=$(blnk_balance "$FEES_ID")

step "Payment and settlement"
pay
ship
out=$(req POST "/orders/$ORDER/confirm-delivery" "$BUYER")
expect 200 "$out" "buyer confirms delivery"
wait_for "merchant credited 4,750 + 150 delivery = 4,900.00 ETB" "blnk_balance $WALLET" "490000"
wait_for "revenue gains the 250.00 ETB fee minus Chapa's 180.25 ETB charge" \
	"echo \$(( \$(blnk_balance $REVENUE_ID) - $REV_BEFORE ))" "6975"
wait_for "Chapa charge booked to processing fees" "echo \$(( \$(blnk_balance $FEES_ID) - $FEES_BEFORE ))" "18025"

step "Dispute and refund"
checkout 1 "second-$RUN"
pay
ship
out=$(req POST "/orders/$ORDER/dispute" "$BUYER" '{"reason":"The package never arrived"}')
expect 200 "$out" "buyer opens dispute"
out=$(req POST "/orders/$ORDER/confirm-delivery" "$BUYER")
expect 400 "$out" "disputed order cannot be confirmed"
out=$(req POST "/admin/orders/$ORDER/resolve" "$ADMIN" '{"outcome":"refund","restock":true}')
expect 200 "$out" "admin resolves the dispute with a refund"
wait_for "order refunded" "field /orders/$ORDER '$BUYER' .order.status" "refunded"
[ "$(blnk_balance "$WALLET")" = "490000" ] || fail "merchant wallet untouched by refunded order"
pass "merchant wallet untouched by the refunded order"

step "Payout"
PAYOUT_BODY='{"amount_cents":400000,"method":"telebirr","destination":"+251 91 123 4567","account_name":"Tigist Bekele","bank_code":"855"}'
out=$(req POST /merchant/wallet/payout "$SELLER" "$PAYOUT_BODY" "Idempotency-Key: kyc-$RUN")
expect 403 "$out" "payout blocked until KYC and shop verification"
out=$(req PUT "/admin/users/$ID_seller/kyc" "$ADMIN" '{"kyc_status":"verified"}')
expect 200 "$out" "admin verifies seller KYC"
out=$(req PUT "/admin/merchants/$MERCHANT_ID/verify" "$ADMIN" '{"is_verified":true}')
expect 200 "$out" "admin verifies the shop"
out=$(req POST /merchant/wallet/payout "$SELLER" '{"amount_cents":400000,"method":"telebirr","destination":"12345","account_name":"Tigist Bekele","bank_code":"855"}' "Idempotency-Key: badphone-$RUN")
expect 400 "$out" "telebirr destination must be an Ethiopian mobile number"
out=$(req POST /merchant/wallet/payout "$SELLER" "$PAYOUT_BODY" "Idempotency-Key: payout-$RUN")
expect 201 "$out" "request 4,000.00 ETB to Telebirr"
PAYOUT=$(echo "$BODY" | jq -r .id)
[ "$(echo "$BODY" | jq -r .destination)" = "0911234567" ] || fail "destination stored in Chapa format"
pass "destination normalized to 0911234567"
out=$(req GET /merchant/dashboard "$SELLER")
expect 200 "$out" "merchant dashboard"
[ "$(echo "$BODY" | jq -r .available_cents)" = "90000" ] || fail "dashboard shows 900 ETB available while 4,000 is held"
pass "dashboard: 900.00 ETB available, 4,000.00 ETB held"
out=$(req POST "/admin/payouts/$PAYOUT/process" "$ADMIN")
expect 202 "$out" "admin approves the payout"
# The payout job starts the Chapa transfer, then re-checks it two minutes later.
wait_for "payout completed" "field /merchant/wallet/payouts/$PAYOUT '$SELLER' .status" "completed" 180
wait_for "merchant wallet now 900.00 ETB" "blnk_balance $WALLET" "90000"

step "Reconciliation"
out=$(req GET /admin/reconciliation "$ADMIN")
expect 200 "$out" "reconciliation report"
MISMATCHES=$(echo "$BODY" | jq '[.mismatches[] | select(.balance_id == "'"$WALLET"'")] | length')
[ "$MISMATCHES" = "0" ] || fail "this merchant's wallet reconciles"
pass "merchant wallet matches its wallet_transactions"

printf '\n\033[32mAll money-flow checks passed.\033[0m\n'
