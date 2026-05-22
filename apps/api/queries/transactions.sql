-- name: CreateTransaction :one
INSERT INTO transactions (type, amount_cents, category_id, member_id, transaction_date, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = $1;

-- name: ListTransactionsByMonth :many
SELECT
    t.id,
    t.type,
    t.amount_cents,
    t.transaction_date,
    t.note,
    t.created_at,
    t.updated_at,
    c.id AS category_id,
    c.name AS category_name,
    c.type AS category_type,
    c.icon_key AS category_icon_key,
    c.color_key AS category_color_key,
    m.id AS member_id,
    m.name AS member_name,
    m.role AS member_role
FROM transactions t
JOIN categories c ON c.id = t.category_id
JOIN members m ON m.id = t.member_id
WHERE t.transaction_date >= $1 AND t.transaction_date < $2
ORDER BY t.transaction_date DESC, t.created_at DESC;

-- name: UpdateTransaction :one
UPDATE transactions
SET
    type = $2,
    amount_cents = $3,
    category_id = $4,
    transaction_date = $5,
    note = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;
