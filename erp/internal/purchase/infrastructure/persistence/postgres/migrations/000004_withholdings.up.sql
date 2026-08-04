CREATE TABLE IF NOT EXISTS purchase.purchase_withholdings (
    id               UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase.orders(id),
    concept_code     VARCHAR(20) NOT NULL,
    concept_name     TEXT NOT NULL,
    base             NUMERIC(18,4) NOT NULL,
    rate_bp          INT NOT NULL,
    amount           NUMERIC(18,4) NOT NULL,
    account_payable  VARCHAR(20) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_withholdings_order ON purchase.purchase_withholdings(purchase_order_id);
