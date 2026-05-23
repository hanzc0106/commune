-- name: GetMonthlyTotals :one
SELECT
    COALESCE(SUM(amount_cents) FILTER (WHERE type = 'income'), 0)::bigint AS income_cents,
    COALESCE(SUM(amount_cents) FILTER (WHERE type = 'expense'), 0)::bigint AS expense_cents
FROM transactions
WHERE transaction_date >= sqlc.arg(start_date) AND transaction_date < sqlc.arg(end_date);

-- name: ListMonthlyExpenseCategoryTotals :many
SELECT
    c.id AS category_id,
    c.name AS category_name,
    c.icon_key AS category_icon_key,
    c.color_key AS category_color_key,
    COALESCE(SUM(t.amount_cents), 0)::bigint AS expense_cents
FROM transactions t
JOIN categories c ON c.id = t.category_id
WHERE t.type = 'expense' AND t.transaction_date >= sqlc.arg(start_date) AND t.transaction_date < sqlc.arg(end_date)
GROUP BY c.id, c.name, c.icon_key, c.color_key
ORDER BY expense_cents DESC, c.name;
