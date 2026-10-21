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
    id, merchant_id, amount_cents, currency, method, destination, status,
    account_name, bank_code, blnk_transaction_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: UpdatePayoutStatus :one
UPDATE payouts
SET
    status = sqlc.arg(status),
    blnk_transaction_id = COALESCE(sqlc.narg(blnk_transaction_id), blnk_transaction_id),
    failure_reason = sqlc.narg(failure_reason),
    processed_at = sqlc.narg(processed_at)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ClaimPayoutForProcessing :one
UPDATE payouts
SET status = 'processing'
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: UpdatePayoutTransfer :one
UPDATE payouts
SET
    transfer_reference = sqlc.arg(transfer_reference),
    provider_response = sqlc.narg(provider_response)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: CountPayoutsByMerchantID :one
SELECT COUNT(*) FROM payouts WHERE merchant_id = $1;

-- name: CountPayoutsByStatus :one
SELECT COUNT(*) FROM payouts WHERE status = $1;

-- name: SumPayoutsSince :one
SELECT COALESCE(SUM(amount_cents), 0)::bigint
FROM payouts
WHERE merchant_id = $1 AND requested_at >= $2 AND status <> 'failed';
