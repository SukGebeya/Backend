-- +goose Up
ALTER TABLE payouts
    ADD COLUMN account_name       TEXT,
    ADD COLUMN bank_code          TEXT,
    ADD COLUMN transfer_reference TEXT UNIQUE,
    ADD COLUMN provider_response  JSONB;

-- +goose Down
ALTER TABLE payouts
    DROP COLUMN IF EXISTS provider_response,
    DROP COLUMN IF EXISTS transfer_reference,
    DROP COLUMN IF EXISTS bank_code,
    DROP COLUMN IF EXISTS account_name;
