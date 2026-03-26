-- name: GetShipmentByID :one
SELECT id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at
FROM shipments
WHERE id = sqlc.arg(id);

-- name: GetShipmentByOrderID :one
SELECT id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at
FROM shipments
WHERE order_id = sqlc.arg(order_id);

-- name: CreateShipment :one
INSERT INTO shipments (id, order_id, status, carrier, tracking_number, estimated_delivery)
VALUES (sqlc.arg(id), sqlc.arg(order_id), sqlc.arg(status), sqlc.arg(carrier), sqlc.arg(tracking_number), sqlc.arg(estimated_delivery))
RETURNING id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at;

-- name: UpdateShipmentStatus :one
UPDATE shipments
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at;

-- name: UpdateShipmentShipped :one
UPDATE shipments
SET status = 'shipped', carrier = sqlc.arg(carrier), tracking_number = sqlc.arg(tracking_number), 
    estimated_delivery = sqlc.arg(estimated_delivery), shipped_at = NOW(), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at;

-- name: UpdateShipmentDelivered :one
UPDATE shipments
SET status = 'delivered', delivered_at = NOW(), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, order_id, status, carrier, tracking_number, estimated_delivery, shipped_at, delivered_at, created_at, updated_at;