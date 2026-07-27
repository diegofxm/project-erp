CREATE SCHEMA IF NOT EXISTS purchase;

CREATE TABLE IF NOT EXISTS purchase.orders (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    supplier_id UUID NOT NULL,
    number      TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date  TIMESTAMPTZ NOT NULL,
    due_date    TIMESTAMPTZ,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_company ON purchase.orders(company_id);

CREATE TABLE IF NOT EXISTS purchase.order_lines (
    id               UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase.orders(id) ON DELETE CASCADE,
    product_id       UUID NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    quantity         NUMERIC(18,4) NOT NULL,
    unit_price       NUMERIC(18,4) NOT NULL,
    tax_rate         NUMERIC(6,2) NOT NULL DEFAULT 0,
    subtotal         NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount       NUMERIC(18,4) NOT NULL DEFAULT 0,
    total            NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_purchase_lines_order ON purchase.order_lines(purchase_order_id);
