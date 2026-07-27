CREATE SCHEMA IF NOT EXISTS sales;

CREATE TABLE IF NOT EXISTS sales.sales (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    customer_id UUID NOT NULL,
    number      TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date  TIMESTAMPTZ NOT NULL,
    due_date    TIMESTAMPTZ,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_company ON sales.sales(company_id);

CREATE TABLE IF NOT EXISTS sales.sale_lines (
    id          UUID PRIMARY KEY,
    sale_id     UUID NOT NULL REFERENCES sales.sales(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity    NUMERIC(18,4) NOT NULL,
    unit_price  NUMERIC(18,4) NOT NULL,
    tax_rate    NUMERIC(6,2) NOT NULL DEFAULT 0,
    subtotal    NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount  NUMERIC(18,4) NOT NULL DEFAULT 0,
    total       NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sale_lines_sale ON sales.sale_lines(sale_id);
