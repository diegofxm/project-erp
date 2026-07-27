CREATE SCHEMA IF NOT EXISTS inventory;

-- Stock actual por producto y bodega
CREATE TABLE IF NOT EXISTS inventory.stock (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    product_id  UUID NOT NULL,
    warehouse   TEXT NOT NULL DEFAULT 'principal',
    quantity    NUMERIC(18,4) NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, product_id, warehouse)
);

CREATE INDEX IF NOT EXISTS idx_stock_company ON inventory.stock(company_id);

-- Historial de movimientos
CREATE TABLE IF NOT EXISTS inventory.movements (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    product_id  UUID NOT NULL,
    warehouse   TEXT NOT NULL DEFAULT 'principal',
    type        VARCHAR(20) NOT NULL,  -- entry, exit, transfer, adjust
    quantity    NUMERIC(18,4) NOT NULL,
    reference   TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_movements_company    ON inventory.movements(company_id);
CREATE INDEX IF NOT EXISTS idx_movements_product    ON inventory.movements(company_id, product_id);
