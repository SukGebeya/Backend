-- +goose Up
CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    provider            TEXT NOT NULL DEFAULT 'chapa',
    status              TEXT NOT NULL DEFAULT 'pending',
    amount_cents        BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    provider_tx_ref     TEXT UNIQUE,
    provider_response   JSONB,
    blnk_transaction_id TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_provider_tx_ref ON payments(provider_tx_ref);
CREATE INDEX idx_payments_status ON payments(status);

-- +goose Down
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_provider_tx_ref;
DROP INDEX IF EXISTS idx_payments_order_id;
DROP TABLE IF EXISTS payments;