-- +goose Up
-- +goose StatementBegin

CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL UNIQUE REFERENCES product_variants(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 0,
    reserved INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_quantity_positive CHECK (quantity >= 0),
    CONSTRAINT chk_reserved_positive CHECK (reserved >= 0),
    CONSTRAINT chk_reserved_lte_quantity CHECK (reserved <= quantity)
);

CREATE INDEX idx_inventory_variant_id ON inventory(variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS inventory;

-- +goose StatementEnd