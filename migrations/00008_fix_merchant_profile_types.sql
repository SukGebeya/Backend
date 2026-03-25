-- +goose Up
-- +goose StatementBegin

ALTER TABLE merchant_profiles
    ALTER COLUMN shop_name TYPE TEXT,
    ALTER COLUMN shop_slug TYPE TEXT,
    ALTER COLUMN contact_phone TYPE TEXT,
    ALTER COLUMN contact_email TYPE TEXT,
    ALTER COLUMN city TYPE TEXT,
    ALTER COLUMN blnk_balance_id TYPE TEXT,
    ALTER COLUMN blnk_ledger_id TYPE TEXT;

ALTER TABLE users
    ALTER COLUMN email TYPE TEXT,
    ALTER COLUMN phone TYPE TEXT,
    ALTER COLUMN password_hash TYPE TEXT,
    ALTER COLUMN role TYPE TEXT,
    ALTER COLUMN kyc_status TYPE TEXT;

ALTER TABLE notes
    ALTER COLUMN title TYPE TEXT;

ALTER TABLE categories
    ALTER COLUMN name TYPE TEXT,
    ALTER COLUMN slug TYPE TEXT;

ALTER TABLE products
    ALTER COLUMN name TYPE TEXT,
    ALTER COLUMN slug TYPE TEXT,
    ALTER COLUMN status TYPE TEXT;

ALTER TABLE product_variants
    ALTER COLUMN sku TYPE TEXT,
    ALTER COLUMN title TYPE TEXT,
    ALTER COLUMN currency TYPE TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE merchant_profiles
    ALTER COLUMN shop_name TYPE VARCHAR(255),
    ALTER COLUMN shop_slug TYPE VARCHAR(255),
    ALTER COLUMN contact_phone TYPE VARCHAR(20),
    ALTER COLUMN contact_email TYPE VARCHAR(255),
    ALTER COLUMN city TYPE VARCHAR(100),
    ALTER COLUMN blnk_balance_id TYPE VARCHAR(255),
    ALTER COLUMN blnk_ledger_id TYPE VARCHAR(255);

ALTER TABLE users
    ALTER COLUMN email TYPE VARCHAR(255),
    ALTER COLUMN phone TYPE VARCHAR(20),
    ALTER COLUMN password_hash TYPE VARCHAR(255),
    ALTER COLUMN role TYPE VARCHAR(50),
    ALTER COLUMN kyc_status TYPE VARCHAR(20);

ALTER TABLE notes
    ALTER COLUMN title TYPE VARCHAR(255);

ALTER TABLE categories
    ALTER COLUMN name TYPE VARCHAR(255),
    ALTER COLUMN slug TYPE VARCHAR(255);

ALTER TABLE products
    ALTER COLUMN name TYPE VARCHAR(255),
    ALTER COLUMN slug TYPE VARCHAR(255),
    ALTER COLUMN status TYPE VARCHAR(20);

ALTER TABLE product_variants
    ALTER COLUMN sku TYPE VARCHAR(255),
    ALTER COLUMN title TYPE VARCHAR(255),
    ALTER COLUMN currency TYPE VARCHAR(10);

-- +goose StatementEnd