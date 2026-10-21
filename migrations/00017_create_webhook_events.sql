-- +goose Up
CREATE TABLE webhook_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    tx_ref          TEXT,
    payload_hash    TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'received',
    error           TEXT,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ,
    UNIQUE (provider, payload_hash)
);

-- status values: 'received', 'processed', 'ignored', 'failed'

CREATE INDEX idx_webhook_events_tx_ref ON webhook_events(tx_ref);
CREATE INDEX idx_webhook_events_status ON webhook_events(status);

-- +goose Down
DROP TABLE IF EXISTS webhook_events;
