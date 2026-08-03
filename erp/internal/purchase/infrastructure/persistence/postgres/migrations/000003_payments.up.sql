CREATE TABLE IF NOT EXISTS purchase.purchase_payments (
    id             UUID PRIMARY KEY,
    company_id     UUID NOT NULL,
    purchase_id    UUID NOT NULL REFERENCES purchase.orders(id),
    payment_date   TIMESTAMPTZ NOT NULL,
    amount         NUMERIC(18,4) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'transfer',
    reference      TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_payments_purchase ON purchase.purchase_payments(purchase_id);
CREATE INDEX IF NOT EXISTS idx_purchase_payments_company  ON purchase.purchase_payments(company_id);
