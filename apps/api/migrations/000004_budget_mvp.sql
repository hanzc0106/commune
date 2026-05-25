CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    month TEXT NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    amount_cents BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT budgets_month_category_unique UNIQUE (month, category_id),
    CONSTRAINT budgets_month_format CHECK (month ~ '^[0-9]{4}-[0-9]{2}$'),
    CONSTRAINT budgets_amount_positive CHECK (amount_cents > 0)
);

CREATE INDEX IF NOT EXISTS budgets_month_idx ON budgets (month);
CREATE INDEX IF NOT EXISTS budgets_category_id_idx ON budgets (category_id);
