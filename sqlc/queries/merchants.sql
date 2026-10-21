-- name: GetMerchantProfileByID :one
SELECT id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
FROM merchant_profiles
WHERE id = sqlc.arg(id);

-- name: GetMerchantProfileByUserID :one
SELECT id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
FROM merchant_profiles
WHERE user_id = sqlc.arg(user_id);

-- name: GetMerchantProfileBySlug :one
SELECT id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
FROM merchant_profiles
WHERE shop_slug = sqlc.arg(shop_slug);

-- name: CreateMerchantProfile :one
INSERT INTO merchant_profiles (
    id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url,
    contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id,
    delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(shop_name), sqlc.arg(shop_slug), sqlc.arg(shop_description), sqlc.arg(shop_logo_url), sqlc.arg(shop_banner_url), sqlc.arg(contact_phone), sqlc.arg(contact_email), sqlc.arg(address), sqlc.arg(city), sqlc.arg(blnk_balance_id), sqlc.arg(blnk_ledger_id),
        sqlc.arg(delivery_fee_cents), sqlc.narg(free_delivery_threshold_cents), sqlc.narg(tin), sqlc.narg(trade_license_number))
RETURNING id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number;

-- name: UpdateMerchantProfile :one
UPDATE merchant_profiles
SET 
    shop_name = COALESCE(sqlc.arg(shop_name), shop_name),
    shop_description = COALESCE(sqlc.arg(shop_description), shop_description),
    shop_logo_url = COALESCE(sqlc.arg(shop_logo_url), shop_logo_url),
    shop_banner_url = COALESCE(sqlc.arg(shop_banner_url), shop_banner_url),
    contact_phone = COALESCE(sqlc.arg(contact_phone), contact_phone),
    contact_email = COALESCE(sqlc.arg(contact_email), contact_email),
    address = COALESCE(sqlc.arg(address), address),
    city = COALESCE(sqlc.arg(city), city),
    blnk_balance_id = COALESCE(sqlc.arg(blnk_balance_id), blnk_balance_id),
    blnk_ledger_id = COALESCE(sqlc.arg(blnk_ledger_id), blnk_ledger_id),
    is_verified = COALESCE(sqlc.arg(is_verified), is_verified),
    delivery_fee_cents = COALESCE(sqlc.narg(delivery_fee_cents), delivery_fee_cents),
    free_delivery_threshold_cents = CASE WHEN sqlc.arg(clear_free_delivery_threshold)::boolean THEN NULL
        ELSE COALESCE(sqlc.narg(free_delivery_threshold_cents), free_delivery_threshold_cents) END,
    tin = COALESCE(sqlc.narg(tin), tin),
    trade_license_number = COALESCE(sqlc.narg(trade_license_number), trade_license_number),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number;

-- name: UpdateMerchantBlnkWallet :one
UPDATE merchant_profiles
SET blnk_balance_id = sqlc.arg(blnk_balance_id), blnk_ledger_id = sqlc.arg(blnk_ledger_id), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number;

-- name: ListMerchantsWithWallets :many
SELECT id, blnk_balance_id
FROM merchant_profiles
WHERE blnk_balance_id IS NOT NULL;

-- name: UpdateMerchantTotalSales :exec
UPDATE merchant_profiles
SET total_sales_cents = total_sales_cents + sqlc.arg(amount), updated_at = NOW()
WHERE id = sqlc.arg(id);

-- name: ListMerchantProfiles :many
SELECT id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
FROM merchant_profiles
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListVerifiedMerchants :many
SELECT id, user_id, shop_name, shop_slug, shop_description, shop_logo_url, shop_banner_url, contact_phone, contact_email, address, city, blnk_balance_id, blnk_ledger_id, is_verified, total_sales_cents, created_at, updated_at, delivery_fee_cents, free_delivery_threshold_cents, tin, trade_license_number
FROM merchant_profiles
WHERE is_verified = true
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);