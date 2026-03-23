-- +goose Up
-- +goose StatementBegin

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS full_name TEXT,
    ADD COLUMN IF NOT EXISTS avatar_url TEXT,
    ADD COLUMN IF NOT EXISTS kyc_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS national_id TEXT,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_users_kyc_status ON users(kyc_status);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
    DROP COLUMN IF EXISTS full_name,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS kyc_status,
    DROP COLUMN IF EXISTS national_id,
    DROP COLUMN IF EXISTS is_active;

-- +goose StatementEnd