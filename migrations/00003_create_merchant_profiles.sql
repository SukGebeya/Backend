-- +goose Up
-- +goose StatementBegin

CREATE TABLE merchant_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    shop_name VARCHAR(255) NOT NULL,
    shop_slug VARCHAR(255) NOT NULL UNIQUE,
    shop_description TEXT,
    shop_logo_url TEXT,
    shop_banner_url TEXT,
    contact_phone VARCHAR(20) NOT NULL,
    contact_email VARCHAR(255),
    address TEXT,
    city VARCHAR(100) NOT NULL DEFAULT 'Addis Ababa',
    blnk_balance_id VARCHAR(255) UNIQUE,
    blnk_ledger_id VARCHAR(255),
    is_verified BOOLEAN NOT NULL DEFAULT false,
    total_sales_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_merchant_profiles_user_id ON merchant_profiles(user_id);
CREATE INDEX idx_merchant_profiles_shop_slug ON merchant_profiles(shop_slug);
CREATE INDEX idx_merchant_profiles_is_verified ON merchant_profiles(is_verified);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS merchant_profiles;

-- +goose StatementEnd