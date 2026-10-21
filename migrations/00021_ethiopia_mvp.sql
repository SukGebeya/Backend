-- +goose Up
ALTER TABLE users
    ADD COLUMN phone_verified_at TIMESTAMPTZ;

ALTER TABLE merchant_profiles
    ADD COLUMN delivery_fee_cents            BIGINT NOT NULL DEFAULT 0 CHECK (delivery_fee_cents >= 0),
    ADD COLUMN free_delivery_threshold_cents BIGINT CHECK (free_delivery_threshold_cents IS NULL OR free_delivery_threshold_cents >= 0),
    ADD COLUMN tin                           TEXT,
    ADD COLUMN trade_license_number          TEXT;

ALTER TABLE orders
    ADD COLUMN delivery_fee_cents BIGINT NOT NULL DEFAULT 0 CHECK (delivery_fee_cents >= 0),
    ADD COLUMN disputed_at        TIMESTAMPTZ,
    ADD COLUMN dispute_reason     TEXT;

ALTER TABLE payments
    ADD COLUMN charge_cents BIGINT NOT NULL DEFAULT 0;

-- Normalize Ethiopian mobile numbers already stored in local or bare formats to +2519XXXXXXXX / +2517XXXXXXXX.
UPDATE users SET phone = '+251' || substring(regexp_replace(phone, '[^0-9]', '', 'g') from 2)
WHERE phone IS NOT NULL AND regexp_replace(phone, '[^0-9]', '', 'g') ~ '^0[79][0-9]{8}$'
  AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone = '+251' || substring(regexp_replace(users.phone, '[^0-9]', '', 'g') from 2));

UPDATE users SET phone = '+' || regexp_replace(phone, '[^0-9]', '', 'g')
WHERE phone IS NOT NULL AND regexp_replace(phone, '[^0-9]', '', 'g') ~ '^251[79][0-9]{8}$' AND phone !~ '^\+251[79][0-9]{8}$'
  AND NOT EXISTS (SELECT 1 FROM users u2 WHERE u2.phone = '+' || regexp_replace(users.phone, '[^0-9]', '', 'g'));

-- +goose Down
ALTER TABLE payments DROP COLUMN IF EXISTS charge_cents;
ALTER TABLE orders
    DROP COLUMN IF EXISTS dispute_reason,
    DROP COLUMN IF EXISTS disputed_at,
    DROP COLUMN IF EXISTS delivery_fee_cents;
ALTER TABLE merchant_profiles
    DROP COLUMN IF EXISTS trade_license_number,
    DROP COLUMN IF EXISTS tin,
    DROP COLUMN IF EXISTS free_delivery_threshold_cents,
    DROP COLUMN IF EXISTS delivery_fee_cents;
ALTER TABLE users DROP COLUMN IF EXISTS phone_verified_at;
