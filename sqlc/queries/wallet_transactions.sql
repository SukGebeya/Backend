-- name: GetWalletTransactionByID :one
SELECT id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description, created_at
FROM wallet_transactions
WHERE id = sqlc.arg(id);

-- name: GetWalletTransactionByOrderID :one
SELECT id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description, created_at
FROM wallet_transactions
WHERE order_id = sqlc.arg(order_id);

-- name: ListWalletTransactionsByMerchantID :many
SELECT id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description, created_at
FROM wallet_transactions
WHERE merchant_id = sqlc.arg(merchant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CreateWalletTransaction :one
INSERT INTO wallet_transactions (id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description)
VALUES (sqlc.arg(id), sqlc.arg(merchant_id), sqlc.arg(order_id), sqlc.arg(type), sqlc.arg(amount_cents), sqlc.arg(currency), 
        sqlc.arg(blnk_transaction_id), sqlc.arg(status), sqlc.arg(description))
RETURNING id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description, created_at;

-- name: UpdateWalletTransactionStatus :one
UPDATE wallet_transactions
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id)
RETURNING id, merchant_id, order_id, type, amount_cents, currency, blnk_transaction_id, status, description, created_at;

-- name: CountWalletTransactionsByMerchantID :one
SELECT COUNT(*) FROM wallet_transactions WHERE merchant_id = sqlc.arg(merchant_id);