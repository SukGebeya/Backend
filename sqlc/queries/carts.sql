-- name: GetCartByID :one
SELECT id, user_id, session_id, status, expires_at, created_at, updated_at
FROM carts
WHERE id = sqlc.arg(id);

-- name: GetCartByUserID :one
SELECT id, user_id, session_id, status, expires_at, created_at, updated_at
FROM carts
WHERE user_id = sqlc.arg(user_id) AND status = 'active';

-- name: GetCartBySessionID :one
SELECT id, user_id, session_id, status, expires_at, created_at, updated_at
FROM carts
WHERE session_id = sqlc.arg(session_id) AND status = 'active';

-- name: CreateCart :one
INSERT INTO carts (id, user_id, session_id, status, expires_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(session_id), sqlc.arg(status), sqlc.arg(expires_at))
RETURNING id, user_id, session_id, status, expires_at, created_at, updated_at;

-- name: UpdateCartStatus :one
UPDATE carts
SET status = sqlc.arg(status), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, user_id, session_id, status, expires_at, created_at, updated_at;

-- name: UpdateCartExpiresAt :one
UPDATE carts
SET expires_at = sqlc.arg(expires_at), updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, user_id, session_id, status, expires_at, created_at, updated_at;

-- name: DeleteCart :exec
DELETE FROM carts WHERE id = sqlc.arg(id);

-- name: DeleteCartBySessionID :exec
DELETE FROM carts WHERE session_id = sqlc.arg(session_id);

-- name: GetCartItemByID :one
SELECT id, cart_id, variant_id, quantity, price_snapshot_cents, added_at
FROM cart_items
WHERE id = sqlc.arg(id);

-- name: GetCartItemByCartAndVariant :one
SELECT id, cart_id, variant_id, quantity, price_snapshot_cents, added_at
FROM cart_items
WHERE cart_id = sqlc.arg(cart_id) AND variant_id = sqlc.arg(variant_id);

-- name: ListCartItems :many
SELECT id, cart_id, variant_id, quantity, price_snapshot_cents, added_at
FROM cart_items
WHERE cart_id = sqlc.arg(cart_id)
ORDER BY added_at ASC;

-- name: CreateCartItem :one
INSERT INTO cart_items (id, cart_id, variant_id, quantity, price_snapshot_cents)
VALUES (sqlc.arg(id), sqlc.arg(cart_id), sqlc.arg(variant_id), sqlc.arg(quantity), sqlc.arg(price_snapshot_cents))
RETURNING id, cart_id, variant_id, quantity, price_snapshot_cents, added_at;

-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET quantity = sqlc.arg(quantity)
WHERE id = sqlc.arg(id)
RETURNING id, cart_id, variant_id, quantity, price_snapshot_cents, added_at;

-- name: DeleteCartItem :exec
DELETE FROM cart_items WHERE id = sqlc.arg(id);

-- name: DeleteCartItemByCartAndVariant :exec
DELETE FROM cart_items WHERE cart_id = sqlc.arg(cart_id) AND variant_id = sqlc.arg(variant_id);

-- name: ClearCartItems :exec
DELETE FROM cart_items WHERE cart_id = sqlc.arg(cart_id);

-- name: CountCartItems :one
SELECT COUNT(*) FROM cart_items WHERE cart_id = sqlc.arg(cart_id);

-- name: CountDistinctVariantsInCart :one
SELECT COUNT(DISTINCT variant_id)::int FROM cart_items WHERE cart_id = sqlc.arg(cart_id);

-- name: UpsertCartItem :one
INSERT INTO cart_items (id, cart_id, variant_id, quantity, price_snapshot_cents)
VALUES (sqlc.arg(id), sqlc.arg(cart_id), sqlc.arg(variant_id), sqlc.arg(quantity), sqlc.arg(price_snapshot_cents))
ON CONFLICT (cart_id, variant_id) 
DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity
RETURNING id, cart_id, variant_id, quantity, price_snapshot_cents, added_at;

-- name: MergeCartItems :exec
INSERT INTO cart_items (id, cart_id, variant_id, quantity, price_snapshot_cents)
SELECT sqlc.arg(new_id), sqlc.arg(target_cart_id), ci.variant_id, ci.quantity, ci.price_snapshot_cents
FROM cart_items ci
WHERE ci.cart_id = sqlc.arg(source_cart_id)
ON CONFLICT (cart_id, variant_id)
DO UPDATE SET quantity = GREATEST(cart_items.quantity, EXCLUDED.quantity);