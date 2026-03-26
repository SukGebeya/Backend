-- +goose Up
-- +goose StatementBegin

CREATE TYPE shipment_status AS ENUM (
    'preparing',
    'shipped',
    'out_for_delivery',
    'delivered',
    'failed'
);

CREATE TABLE shipments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id           UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    status              shipment_status NOT NULL DEFAULT 'preparing',
    carrier             TEXT,
    tracking_number     TEXT,
    estimated_delivery  DATE,
    shipped_at          TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_shipments_order_id ON shipments(order_id);
CREATE INDEX idx_shipments_status ON shipments(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_shipments_status;
DROP INDEX IF EXISTS idx_shipments_order_id;
DROP TABLE IF EXISTS shipments;
DROP TYPE IF EXISTS shipment_status;

-- +goose StatementEnd