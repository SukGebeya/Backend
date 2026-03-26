-- +goose Up
CREATE TABLE payouts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id         UUID NOT NULL REFERENCES merchant_profiles(id) ON DELETE CASCADE,
    amount_cents        BIGINT NOT NULL,
    currency            TEXT NOT NULL DEFAULT 'ETB',
    method              TEXT NOT NULL,
    destination         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending',
    blnk_transaction_id TEXT,
    failure_reason      TEXT,
    requested_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at        TIMESTAMPTZ
);

CREATE INDEX idx_payouts_merchant_id ON payouts(merchant_id);
CREATE INDEX idx_payouts_status ON payouts(status);
CREATE INDEX idx_payouts_requested_at ON payouts(requested_at DESC);

-- +goose Down
DROP TABLE IF EXISTS payouts;