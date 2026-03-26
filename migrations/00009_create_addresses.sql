-- +goose Up
-- +goose StatementBegin

CREATE TABLE addresses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name       TEXT NOT NULL,
    phone           TEXT NOT NULL,
    line1           TEXT NOT NULL,
    line2           TEXT,
    city            TEXT NOT NULL DEFAULT 'Addis Ababa',
    subcity         TEXT,
    woreda          TEXT,
    is_default      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS addresses;

-- +goose StatementEnd