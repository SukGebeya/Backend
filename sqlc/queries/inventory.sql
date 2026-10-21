-- name: GetInventoryByVariantID :one
SELECT id, variant_id, quantity, reserved, updated_at
FROM inventory
WHERE variant_id = sqlc.arg(variant_id);

-- name: GetInventoryByID :one
SELECT id, variant_id, quantity, reserved, updated_at
FROM inventory
WHERE id = sqlc.arg(id);

-- name: CreateInventory :one
INSERT INTO inventory (id, variant_id, quantity, reserved)
VALUES (sqlc.arg(id), sqlc.arg(variant_id), sqlc.arg(quantity), sqlc.arg(reserved))
RETURNING id, variant_id, quantity, reserved, updated_at;

-- name: UpdateInventoryQuantity :one
UPDATE inventory
SET quantity = sqlc.arg(quantity), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, variant_id, quantity, reserved, updated_at;

-- name: ReserveInventory :execrows
UPDATE inventory
SET reserved = reserved + sqlc.arg(qty), updated_at = NOW()
WHERE variant_id = sqlc.arg(variant_id) AND (quantity - reserved) >= sqlc.arg(qty);

-- name: ReleaseInventory :execrows
UPDATE inventory
SET reserved = reserved - sqlc.arg(qty), updated_at = NOW()
WHERE variant_id = sqlc.arg(variant_id) AND reserved >= sqlc.arg(qty);

-- name: CommitInventory :execrows
UPDATE inventory
SET quantity = quantity - sqlc.arg(qty), reserved = reserved - sqlc.arg(qty), updated_at = NOW()
WHERE variant_id = sqlc.arg(variant_id) AND reserved >= sqlc.arg(qty);

-- name: RestockInventory :execrows
UPDATE inventory
SET quantity = quantity + sqlc.arg(qty), updated_at = NOW()
WHERE variant_id = sqlc.arg(variant_id);

-- name: GetInventoryByProductID :many
SELECT i.id, i.variant_id, i.quantity, i.reserved, i.updated_at
FROM inventory i
JOIN product_variants pv ON i.variant_id = pv.id
WHERE pv.product_id = sqlc.arg(product_id);