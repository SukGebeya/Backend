-- name: GetPayoutByID :one
SELECT * FROM payouts WHERE id = $1;

-- name: ListPayoutsByMerchantID :many
SELECT * FROM payouts
WHERE merchant_id = $1
ORDER BY requested_at DESC
OFFSET $2 LIMIT $3;

-- name: ListPayoutsByStatus :many
SELECT * FROM payouts
WHERE status = $1
ORDER BY requested_at DESC
OFFSET $2 LIMIT $3;

-- name: ListAllPayouts :many
SELECT * FROM payouts
ORDER BY requested_at DESC
OFFSET $1 LIMIT $2;

-- name: CreatePayout :one
INSERT INTO payouts (
    id, merchant_id, amount_cents, currency, method, destination, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdatePayoutStatus :one
UPDATE payouts
SET 
    status = $2, 
    blnk_transaction_id = $3, 
    failure_reason = $4, 
    processed_at = $5
WHERE id = $1
RETURNING *;

-- name: CountPayoutsByMerchantID :one
SELECT COUNT(*) FROM payouts WHERE merchant_id = $1;

-- name: CountPayoutsByStatus :one
SELECT COUNT(*) FROM payouts WHERE status = $1;