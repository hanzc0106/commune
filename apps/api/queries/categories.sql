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
