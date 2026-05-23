CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    icon_key TEXT NOT NULL DEFAULT 'circle',
    color_key TEXT NOT NULL DEFAULT 'slate',
    sort_order INTEGER NOT NULL DEFAULT 0,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    system_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT categories_type_valid CHECK (type IN ('expense', 'income'))
);

CREATE INDEX IF NOT EXISTS categories_active_sort_idx ON categories (active, type, sort_order, name);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id),
    member_id UUID NOT NULL REFERENCES members(id),
    transaction_date DATE NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transactions_type_valid CHECK (type IN ('expense', 'income')),
    CONSTRAINT transactions_amount_positive CHECK (amount_cents > 0)
);

CREATE INDEX IF NOT EXISTS transactions_month_idx ON transactions (transaction_date DESC);
CREATE INDEX IF NOT EXISTS transactions_category_id_idx ON transactions (category_id);
CREATE INDEX IF NOT EXISTS transactions_member_id_idx ON transactions (member_id);
