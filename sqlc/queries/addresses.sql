-- name: GetAddressByID :one
SELECT id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at
FROM addresses
WHERE id = sqlc.arg(id);

-- name: ListAddressesByUserID :many
SELECT id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at
FROM addresses
WHERE user_id = sqlc.arg(user_id)
ORDER BY is_default DESC, created_at DESC;

-- name: GetDefaultAddress :one
SELECT id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at
FROM addresses
WHERE user_id = sqlc.arg(user_id) AND is_default = true;

-- name: CreateAddress :one
INSERT INTO addresses (id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(full_name), sqlc.arg(phone), sqlc.arg(line1), sqlc.arg(line2), sqlc.arg(city), sqlc.arg(subcity), sqlc.arg(woreda), sqlc.arg(is_default))
RETURNING id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at;

-- name: UpdateAddress :one
UPDATE addresses
SET 
    full_name = COALESCE(sqlc.arg(full_name), full_name),
    phone = COALESCE(sqlc.arg(phone), phone),
    line1 = COALESCE(sqlc.arg(line1), line1),
    line2 = COALESCE(sqlc.arg(line2), line2),
    city = COALESCE(sqlc.arg(city), city),
    subcity = COALESCE(sqlc.arg(subcity), subcity),
    woreda = COALESCE(sqlc.arg(woreda), woreda),
    is_default = COALESCE(sqlc.arg(is_default), is_default)
WHERE id = sqlc.arg(id)
RETURNING id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at;

-- name: SetDefaultAddress :exec
UPDATE addresses SET is_default = false WHERE user_id = sqlc.arg(user_id);

-- name: SetAddressAsDefault :one
UPDATE addresses SET is_default = true WHERE id = sqlc.arg(id) RETURNING id, user_id, full_name, phone, line1, line2, city, subcity, woreda, is_default, created_at;

-- name: DeleteAddress :exec
DELETE FROM addresses WHERE id = sqlc.arg(id);

-- name: CountAddressesByUser :one
SELECT COUNT(*) FROM addresses WHERE user_id = sqlc.arg(user_id);