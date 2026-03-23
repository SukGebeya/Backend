-- name: GetCategoryByID :one
SELECT id, parent_id, name, slug, icon_url, is_active, sort_order, created_at
FROM categories
WHERE id = sqlc.arg(id);

-- name: GetCategoryBySlug :one
SELECT id, parent_id, name, slug, icon_url, is_active, sort_order, created_at
FROM categories
WHERE slug = sqlc.arg(slug);

-- name: ListCategories :many
SELECT id, parent_id, name, slug, icon_url, is_active, sort_order, created_at
FROM categories
WHERE is_active = true
ORDER BY sort_order ASC, name ASC;

-- name: ListCategoriesByParent :many
SELECT id, parent_id, name, slug, icon_url, is_active, sort_order, created_at
FROM categories
WHERE parent_id = sqlc.arg(parent_id) AND is_active = true
ORDER BY sort_order ASC, name ASC;

-- name: ListTopLevelCategories :many
SELECT id, parent_id, name, slug, icon_url, is_active, sort_order, created_at
FROM categories
WHERE parent_id IS NULL AND is_active = true
ORDER BY sort_order ASC, name ASC;

-- name: CreateCategory :one
INSERT INTO categories (id, parent_id, name, slug, icon_url, is_active, sort_order)
VALUES (sqlc.arg(id), sqlc.arg(parent_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(icon_url), sqlc.arg(is_active), sqlc.arg(sort_order))
RETURNING id, parent_id, name, slug, icon_url, is_active, sort_order, created_at;

-- name: UpdateCategory :one
UPDATE categories
SET 
    name = COALESCE(sqlc.arg(name), name),
    slug = COALESCE(sqlc.arg(slug), slug),
    icon_url = COALESCE(sqlc.arg(icon_url), icon_url),
    is_active = COALESCE(sqlc.arg(is_active), is_active),
    sort_order = COALESCE(sqlc.arg(sort_order), sort_order)
WHERE id = sqlc.arg(id)
RETURNING id, parent_id, name, slug, icon_url, is_active, sort_order, created_at;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = sqlc.arg(id);