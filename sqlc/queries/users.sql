-- name: GetUserByID :one
SELECT id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at
FROM users
WHERE id = sqlc.arg(id);

-- name: GetUserByEmail :one
SELECT id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at
FROM users
WHERE email = sqlc.arg(email);

-- name: GetUserByPhone :one
SELECT id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at
FROM users
WHERE phone = sqlc.arg(phone);

-- name: GetUserByEmailOrPhone :one
SELECT id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at
FROM users
WHERE email = sqlc.arg(identifier) OR phone = sqlc.arg(identifier);

-- name: CreateUser :one
INSERT INTO users (id, email, phone, password_hash, role, kyc_status, is_active)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(phone), sqlc.arg(password_hash), sqlc.arg(role), sqlc.arg(kyc_status), sqlc.arg(is_active))
RETURNING id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET 
    email = COALESCE(sqlc.arg(email), email),
    phone = COALESCE(sqlc.arg(phone), phone),
    password_hash = COALESCE(sqlc.arg(password_hash), password_hash),
    role = COALESCE(sqlc.arg(role), role),
    full_name = COALESCE(sqlc.arg(full_name), full_name),
    avatar_url = COALESCE(sqlc.arg(avatar_url), avatar_url),
    kyc_status = COALESCE(sqlc.arg(kyc_status), kyc_status),
    national_id = COALESCE(sqlc.arg(national_id), national_id),
    is_active = COALESCE(sqlc.arg(is_active), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = sqlc.arg(id);

-- name: ListUsers :many
SELECT id, email, phone, password_hash, role, full_name, avatar_url, kyc_status, national_id, is_active, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;