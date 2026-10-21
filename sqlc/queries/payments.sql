-- name: GetPaymentByID :one
SELECT id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents
FROM payments
WHERE id = sqlc.arg(id);

-- name: GetPaymentByOrderID :one
SELECT id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents
FROM payments
WHERE order_id = sqlc.arg(order_id);

-- name: GetPaymentByProviderTxRef :one
SELECT id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents
FROM payments
WHERE provider_tx_ref = sqlc.arg(provider_tx_ref);

-- name: ListPaymentsByStatus :many
SELECT id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents
FROM payments
WHERE status = sqlc.arg(status)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreatePayment :one
INSERT INTO payments (id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id)
VALUES (sqlc.arg(id), sqlc.arg(order_id), sqlc.arg(provider), sqlc.arg(status), sqlc.arg(amount_cents), sqlc.arg(currency), sqlc.arg(provider_tx_ref), sqlc.arg(provider_response), sqlc.arg(blnk_transaction_id))
RETURNING id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents;

-- name: UpdatePaymentProviderResponse :one
UPDATE payments
SET provider_response = sqlc.arg(provider_response), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents;

-- name: UpdatePaymentBlnkTransaction :one
UPDATE payments
SET blnk_transaction_id = sqlc.arg(blnk_transaction_id), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents;

-- name: UpdatePaymentResult :one
UPDATE payments
SET status = sqlc.arg(status),
    provider_response = COALESCE(sqlc.narg(provider_response), provider_response),
    blnk_transaction_id = COALESCE(sqlc.narg(blnk_transaction_id), blnk_transaction_id),
    charge_cents = COALESCE(sqlc.narg(charge_cents), charge_cents),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, provider, status, amount_cents, currency, provider_tx_ref, provider_response, blnk_transaction_id, created_at, updated_at, charge_cents;
