-- +goose Up
-- +goose StatementBegin

CREATE TABLE cart_items (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id                 UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    variant_id              UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity                INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    price_snapshot_cents    BIGINT NOT NULL,
    added_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(cart_id, variant_id)
);

CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX idx_cart_items_variant_id ON cart_items(variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS cart_items;

-- +goose StatementEnd