-- name: CreateCategory :one
INSERT INTO categories (name, type, icon_key, color_key, sort_order, active, system_default)
VALUES ($1, $2, $3, $4, $5, TRUE, $6)
RETURNING *;

-- name: ListActiveCategories :many
SELECT *
FROM categories
WHERE active = TRUE
ORDER BY type, sort_order, name;

-- name: GetCategoryByID :one
SELECT *
FROM categories
WHERE id = $1;

-- name: ListCategories :many
SELECT *
FROM categories
ORDER BY active DESC, type, sort_order, name;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = $2,
    icon_key = $3,
    color_key = $4,
    sort_order = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DisableCategory :one
UPDATE categories
SET active = FALSE, updated_at = now()
WHERE id = $1
RETURNING *;
