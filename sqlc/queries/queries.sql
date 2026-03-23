-- name: GetUserByID :one
SELECT id, email, phone, password_hash, role, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, phone, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByPhone :one
SELECT id, email, phone, password_hash, role, created_at, updated_at
FROM users
WHERE phone = $1;

-- name: GetUserByEmailOrPhone :one
SELECT id, email, phone, password_hash, role, created_at, updated_at
FROM users
WHERE email = $1 OR phone = $1;

-- name: CreateUser :one
INSERT INTO users (id, email, phone, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, phone, password_hash, role, created_at, updated_at;

-- name: UpdateUser :one
UPDATE users
SET email = $2, phone = $3, password_hash = $4, role = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, email, phone, password_hash, role, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, phone, password_hash, role, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetNoteByID :one
SELECT id, user_id, title, content, created_at, updated_at, deleted_at
FROM notes
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetNoteByIDWithDeleted :one
SELECT id, user_id, title, content, created_at, updated_at, deleted_at
FROM notes
WHERE id = $1;

-- name: GetNoteByIDAndUserID :one
SELECT id, user_id, title, content, created_at, updated_at, deleted_at
FROM notes
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListNotesByUserID :many
SELECT id, user_id, title, content, created_at, updated_at, deleted_at
FROM notes
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListNotesByUserIDWithDeleted :many
SELECT id, user_id, title, content, created_at, updated_at, deleted_at
FROM notes
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateNote :one
INSERT INTO notes (id, user_id, title, content)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, title, content, created_at, updated_at, deleted_at;

-- name: UpdateNote :one
UPDATE notes
SET title = $2, content = $3, updated_at = NOW()
WHERE id = $1 AND user_id = $4 AND deleted_at IS NULL
RETURNING id, user_id, title, content, created_at, updated_at, deleted_at;

-- name: SoftDeleteNote :exec
UPDATE notes
SET deleted_at = NOW()
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: RestoreNote :exec
UPDATE notes
SET deleted_at = NULL, updated_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: HardDeleteNote :exec
DELETE FROM notes WHERE id = $1;

-- name: CountNotesByUserID :one
SELECT COUNT(*) FROM notes WHERE user_id = $1 AND deleted_at IS NULL;