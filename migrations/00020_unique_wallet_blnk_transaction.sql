-- +goose Up
CREATE UNIQUE INDEX idx_wallet_transactions_blnk_txn ON wallet_transactions(blnk_transaction_id);

-- +goose Down
DROP INDEX IF EXISTS idx_wallet_transactions_blnk_txn;
