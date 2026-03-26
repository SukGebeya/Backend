-- +goose Up
-- +goose StatementBegin

CREATE TABLE wallet_transactions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id          UUID NOT NULL REFERENCES merchant_profiles(id) ON DELETE RESTRICT,
    order_id            UUID REFERENCES orders(id) ON DELETE SET NULL,
    type                 TEXT NOT NULL,
    amount_cents         BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    blnk_transaction_id TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'applied',
    description          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- type values: 'sale_credit', 'payout_debit', 'refund_debit'
-- status values: 'inflight', 'applied', 'rejected'

CREATE INDEX idx_wallet_transactions_merchant_id ON wallet_transactions(merchant_id);
CREATE INDEX idx_wallet_transactions_order_id ON wallet_transactions(order_id);
CREATE INDEX idx_wallet_transactions_type ON wallet_transactions(type);
CREATE INDEX idx_wallet_transactions_created_at ON wallet_transactions(created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_wallet_transactions_created_at;
DROP INDEX IF EXISTS idx_wallet_transactions_type;
DROP INDEX IF EXISTS idx_wallet_transactions_order_id;
DROP INDEX IF EXISTS idx_wallet_transactions_merchant_id;
DROP TABLE IF EXISTS wallet_transactions;

-- +goose StatementEnd