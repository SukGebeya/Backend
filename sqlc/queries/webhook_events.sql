-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (id, provider, event_type, tx_ref, payload_hash, payload, status)
VALUES (sqlc.arg(id), sqlc.arg(provider), sqlc.arg(event_type), sqlc.narg(tx_ref), sqlc.arg(payload_hash), sqlc.arg(payload), sqlc.arg(status))
ON CONFLICT (provider, payload_hash) DO NOTHING
RETURNING id, provider, event_type, tx_ref, payload_hash, payload, status, error, received_at, processed_at;

-- name: GetWebhookEventByID :one
SELECT id, provider, event_type, tx_ref, payload_hash, payload, status, error, received_at, processed_at
FROM webhook_events
WHERE id = sqlc.arg(id);

-- name: UpdateWebhookEventStatus :exec
UPDATE webhook_events
SET status = sqlc.arg(status), error = sqlc.narg(error), processed_at = NOW()
WHERE id = sqlc.arg(id);
