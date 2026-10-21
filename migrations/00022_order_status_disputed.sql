-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'disputed';

-- +goose Down
-- Postgres cannot drop enum values; 'disputed' stays defined but unused.
SELECT 1;
