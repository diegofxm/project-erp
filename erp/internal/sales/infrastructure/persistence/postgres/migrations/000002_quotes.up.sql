CREATE TABLE IF NOT EXISTS sales.quotes (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    customer_id UUID NOT NULL,
    number      TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date  TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotes_company ON sales.quotes(company_id);

CREATE TABLE IF NOT EXISTS sales.quote_lines (
    id          UUID PRIMARY KEY,
    quote_id    UUID NOT NULL REFERENCES sales.quotes(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity    NUMERIC(18,4) NOT NULL,
    unit_price  NUMERIC(18,4) NOT NULL,
    tax_rate    NUMERIC(6,2) NOT NULL DEFAULT 0,
    subtotal    NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount  NUMERIC(18,4) NOT NULL DEFAULT 0,
    total       NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_quote_lines_quote ON sales.quote_lines(quote_id);
