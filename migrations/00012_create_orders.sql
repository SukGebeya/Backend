-- +goose Up
CREATE TYPE order_status AS ENUM (
    'pending_payment',
    'paid',
    'processing',
    'shipped',
    'delivered',
    'cancelled',
    'refunded'
);

CREATE TABLE orders (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id                UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    merchant_id             UUID NOT NULL REFERENCES merchant_profiles(id) ON DELETE RESTRICT,
    shipping_address_id     UUID REFERENCES addresses(id) ON DELETE SET NULL,
    status                  order_status NOT NULL DEFAULT 'pending_payment',
    subtotal_cents          BIGINT NOT NULL,
    platform_fee_cents      BIGINT NOT NULL,
    total_cents             BIGINT NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'ETB',
    notes                   TEXT,
    blnk_transaction_id     TEXT,
    chapa_tx_ref            TEXT UNIQUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    variant_id          UUID NOT NULL REFERENCES product_variants(id) ON DELETE RESTRICT,
    product_name        TEXT NOT NULL,
    variant_title       TEXT NOT NULL,
    quantity            INT NOT NULL,
    unit_price_cents    BIGINT NOT NULL,
    total_price_cents   BIGINT NOT NULL
);

CREATE INDEX idx_orders_buyer_id ON orders(buyer_id);
CREATE INDEX idx_orders_merchant_id ON orders(merchant_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_chapa_tx_ref ON orders(chapa_tx_ref);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_variant_id ON order_items(variant_id);

-- +goose Down
DROP INDEX IF EXISTS idx_order_items_variant_id;
DROP INDEX IF EXISTS idx_order_items_order_id;
DROP INDEX IF EXISTS idx_orders_chapa_tx_ref;
DROP INDEX IF EXISTS idx_orders_status;
DROP INDEX IF EXISTS idx_orders_merchant_id;
DROP INDEX IF EXISTS idx_orders_buyer_id;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_status;