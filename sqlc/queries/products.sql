-- name: GetProductByID :one
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE id = sqlc.arg(id);

-- name: GetProductBySlug :one
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE slug = sqlc.arg(slug);

-- name: GetProductBySlugAndMerchant :one
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE slug = sqlc.arg(slug) AND merchant_id = sqlc.arg(merchant_id);

-- name: ListProducts :many
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListProductsByMerchant :many
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE merchant_id = sqlc.arg(merchant_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListProductsByCategory :many
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE category_id = sqlc.arg(category_id) AND status = 'active'
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: ListProductsByMerchantAndStatus :many
SELECT id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at
FROM products
WHERE merchant_id = sqlc.arg(merchant_id) AND status = sqlc.arg(status)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CreateProduct :one
INSERT INTO products (id, merchant_id, category_id, name, slug, description, status, is_digital)
VALUES (sqlc.arg(id), sqlc.arg(merchant_id), sqlc.arg(category_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(description), sqlc.arg(status), sqlc.arg(is_digital))
RETURNING id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at;

-- name: UpdateProduct :one
UPDATE products
SET 
    name = COALESCE(sqlc.arg(name), name),
    category_id = COALESCE(sqlc.arg(category_id), category_id),
    description = COALESCE(sqlc.arg(description), description),
    status = COALESCE(sqlc.arg(status), status),
    is_digital = COALESCE(sqlc.arg(is_digital), is_digital),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, merchant_id, category_id, name, slug, description, status, is_digital, created_at, updated_at;

-- name: DeleteProduct :exec
DELETE FROM products WHERE id = sqlc.arg(id);

-- name: CountProductsByMerchant :one
SELECT COUNT(*) FROM products WHERE merchant_id = sqlc.arg(merchant_id);

-- Product Images

-- name: GetProductImages :many
SELECT id, product_id, url, alt_text, sort_order, is_primary
FROM product_images
WHERE product_id = sqlc.arg(product_id)
ORDER BY sort_order ASC;

-- name: GetProductPrimaryImage :one
SELECT id, product_id, url, alt_text, sort_order, is_primary
FROM product_images
WHERE product_id = sqlc.arg(product_id) AND is_primary = true
LIMIT 1;

-- name: CreateProductImage :one
INSERT INTO product_images (id, product_id, url, alt_text, sort_order, is_primary)
VALUES (sqlc.arg(id), sqlc.arg(product_id), sqlc.arg(url), sqlc.arg(alt_text), sqlc.arg(sort_order), sqlc.arg(is_primary))
RETURNING id, product_id, url, alt_text, sort_order, is_primary;

-- name: UpdateProductImage :one
UPDATE product_images
SET 
    url = COALESCE(sqlc.arg(url), url),
    alt_text = COALESCE(sqlc.arg(alt_text), alt_text),
    sort_order = COALESCE(sqlc.arg(sort_order), sort_order),
    is_primary = COALESCE(sqlc.arg(is_primary), is_primary)
WHERE id = sqlc.arg(id)
RETURNING id, product_id, url, alt_text, sort_order, is_primary;

-- name: DeleteProductImage :exec
DELETE FROM product_images WHERE id = sqlc.arg(id);

-- name: DeleteProductImages :exec
DELETE FROM product_images WHERE product_id = sqlc.arg(product_id);

-- Product Variants

-- name: GetVariantByID :one
SELECT id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at
FROM product_variants
WHERE id = sqlc.arg(id);

-- name: GetVariantBySKU :one
SELECT id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at
FROM product_variants
WHERE sku = sqlc.arg(sku);

-- name: GetVariantsByProduct :many
SELECT id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at
FROM product_variants
WHERE product_id = sqlc.arg(product_id)
ORDER BY created_at ASC;

-- name: GetActiveVariantsByProduct :many
SELECT id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at
FROM product_variants
WHERE product_id = sqlc.arg(product_id) AND is_active = true
ORDER BY created_at ASC;

-- name: CreateVariant :one
INSERT INTO product_variants (id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active)
VALUES (sqlc.arg(id), sqlc.arg(product_id), sqlc.arg(sku), sqlc.arg(title), sqlc.arg(price_cents), sqlc.arg(compare_price_cents), sqlc.arg(currency), sqlc.arg(attributes), sqlc.arg(is_active))
RETURNING id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at;

-- name: UpdateVariant :one
UPDATE product_variants
SET 
    title = COALESCE(sqlc.arg(title), title),
    price_cents = COALESCE(sqlc.arg(price_cents), price_cents),
    compare_price_cents = COALESCE(sqlc.arg(compare_price_cents), compare_price_cents),
    attributes = COALESCE(sqlc.arg(attributes), attributes),
    is_active = COALESCE(sqlc.arg(is_active), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, product_id, sku, title, price_cents, compare_price_cents, currency, attributes, is_active, created_at, updated_at;

-- name: DeleteVariant :exec
DELETE FROM product_variants WHERE id = sqlc.arg(id);

-- name: DeleteVariantsByProduct :exec
DELETE FROM product_variants WHERE product_id = sqlc.arg(product_id);