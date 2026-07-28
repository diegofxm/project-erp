CREATE TABLE IF NOT EXISTS sales.sale_payments (
    id             UUID PRIMARY KEY,
    company_id     UUID NOT NULL,
    sale_id        UUID NOT NULL REFERENCES sales.sales(id),
    payment_date   TIMESTAMPTZ NOT NULL,
    amount         NUMERIC(18,4) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'transfer',
    reference      TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_payments_sale    ON sales.sale_payments(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_payments_company ON sales.sale_payments(company_id);
