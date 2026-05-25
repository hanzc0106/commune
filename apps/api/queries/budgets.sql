-- name: ListBudgetsByMonth :many
SELECT id, month, category_id, amount_cents, created_at, updated_at
FROM budgets
WHERE month = $1
ORDER BY category_id;

-- name: UpsertBudget :one
INSERT INTO budgets (month, category_id, amount_cents)
VALUES ($1, $2, $3)
ON CONFLICT (month, category_id)
DO UPDATE SET amount_cents = EXCLUDED.amount_cents, updated_at = now()
RETURNING id, month, category_id, amount_cents, created_at, updated_at;

-- name: CopyPreviousBudgets :one
WITH inserted AS (
    INSERT INTO budgets (month, category_id, amount_cents)
    SELECT sqlc.arg(target_month)::text, b.category_id, b.amount_cents
    FROM budgets b
    JOIN categories c ON c.id = b.category_id
    WHERE b.month = sqlc.arg(source_month)
      AND c.active = TRUE
      AND c.type = 'expense'
    ON CONFLICT (month, category_id) DO NOTHING
    RETURNING id
)
SELECT count(*)::bigint AS copied_count
FROM inserted;

-- name: ListMonthlyBudgetSpending :many
SELECT
    t.category_id,
    COALESCE(SUM(t.amount_cents), 0)::bigint AS spent_cents
FROM transactions t
WHERE t.type = 'expense'
  AND t.transaction_date >= sqlc.arg(start_date)
  AND t.transaction_date < sqlc.arg(end_date)
GROUP BY t.category_id;
