-- name: GetOrderByID :one
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at
FROM orders
WHERE id = sqlc.arg(id);

-- name: GetOrderByChapaTxRef :one
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at
FROM orders
WHERE chapa_tx_ref = sqlc.arg(chapa_tx_ref);

-- name: ListOrdersByBuyerID :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at
FROM orders
WHERE buyer_id = sqlc.arg(buyer_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListOrdersByMerchantID :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at
FROM orders
WHERE merchant_id = sqlc.arg(merchant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListOrdersByStatus :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at
FROM orders
WHERE status = sqlc.arg(status)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreateOrder :one
INSERT INTO orders (id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, chapa_tx_ref)
VALUES (sqlc.arg(id), sqlc.arg(buyer_id), sqlc.arg(merchant_id), sqlc.arg(shipping_address_id), sqlc.arg(status), sqlc.arg(subtotal_cents), sqlc.arg(platform_fee_cents), sqlc.arg(total_cents), sqlc.arg(currency), sqlc.arg(notes), sqlc.arg(chapa_tx_ref))
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at;

-- name: UpdateOrderBlnkTransaction :one
UPDATE orders
SET blnk_transaction_id = sqlc.arg(blnk_transaction_id), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at;

-- name: CountOrdersByBuyerID :one
SELECT COUNT(*) FROM orders WHERE buyer_id = sqlc.arg(buyer_id);

-- name: CountOrdersByMerchantID :one
SELECT COUNT(*) FROM orders WHERE merchant_id = sqlc.arg(merchant_id);

-- name: GetOrderItemByID :one
SELECT id, order_id, variant_id, product_name, variant_title, quantity, unit_price_cents, total_price_cents
FROM order_items
WHERE id = sqlc.arg(id);

-- name: ListOrderItemsByOrderID :many
SELECT id, order_id, variant_id, product_name, variant_title, quantity, unit_price_cents, total_price_cents
FROM order_items
WHERE order_id = sqlc.arg(order_id)
ORDER BY id ASC;

-- name: CreateOrderItem :one
INSERT INTO order_items (id, order_id, variant_id, product_name, variant_title, quantity, unit_price_cents, total_price_cents)
VALUES (sqlc.arg(id), sqlc.arg(order_id), sqlc.arg(variant_id), sqlc.arg(product_name), sqlc.arg(variant_title), sqlc.arg(quantity), sqlc.arg(unit_price_cents), sqlc.arg(total_price_cents))
RETURNING id, order_id, variant_id, product_name, variant_title, quantity, unit_price_cents, total_price_cents;