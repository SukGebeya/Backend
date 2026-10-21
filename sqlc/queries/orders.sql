-- name: GetOrderByID :one
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason
FROM orders
WHERE id = sqlc.arg(id);

-- name: GetOrderByChapaTxRef :one
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason
FROM orders
WHERE chapa_tx_ref = sqlc.arg(chapa_tx_ref);

-- name: ListOrdersByBuyerID :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason
FROM orders
WHERE buyer_id = sqlc.arg(buyer_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListOrdersByMerchantID :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason
FROM orders
WHERE merchant_id = sqlc.arg(merchant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListOrdersByStatus :many
SELECT id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason
FROM orders
WHERE status = sqlc.arg(status)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreateOrder :one
INSERT INTO orders (id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, chapa_tx_ref, delivery_fee_cents)
VALUES (sqlc.arg(id), sqlc.arg(buyer_id), sqlc.arg(merchant_id), sqlc.arg(shipping_address_id), sqlc.arg(status), sqlc.arg(subtotal_cents), sqlc.arg(platform_fee_cents), sqlc.arg(total_cents), sqlc.arg(currency), sqlc.arg(notes), sqlc.arg(chapa_tx_ref), sqlc.arg(delivery_fee_cents))
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason;

-- name: TransitionOrderStatus :one
UPDATE orders
SET status = sqlc.arg(to_status), updated_at = NOW()
WHERE id = sqlc.arg(id) AND status::text = ANY(sqlc.arg(from_statuses)::text[])
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason;

-- name: UpdateOrderBlnkTransaction :one
UPDATE orders
SET blnk_transaction_id = sqlc.arg(blnk_transaction_id), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason;

-- name: OpenOrderDispute :one
UPDATE orders
SET status = 'disputed', disputed_at = NOW(), dispute_reason = sqlc.arg(reason), updated_at = NOW()
WHERE id = sqlc.arg(id) AND status::text = ANY(sqlc.arg(from_statuses)::text[])
RETURNING id, buyer_id, merchant_id, shipping_address_id, status, subtotal_cents, platform_fee_cents, total_cents, currency, notes, blnk_transaction_id, chapa_tx_ref, created_at, updated_at, delivery_fee_cents, disputed_at, dispute_reason;

-- name: CountOrdersByMerchantGroupedByStatus :many
SELECT status::text AS status, COUNT(*)::bigint AS count
FROM orders
WHERE merchant_id = sqlc.arg(merchant_id)
GROUP BY status;

-- name: SumUnsettledEscrowCents :one
SELECT COALESCE(SUM(o.total_cents), 0)::bigint
FROM orders o
WHERE o.status IN ('paid', 'processing', 'shipped', 'disputed', 'delivered')
  AND NOT EXISTS (SELECT 1 FROM wallet_transactions wt WHERE wt.order_id = o.id AND wt.type = 'sale_credit');

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