-- Bodega por defecto: la que usan las ventas/compras cuando el documento no elige una
-- explícitamente. Un índice único parcial garantiza que solo pueda haber una por empresa.
ALTER TABLE company.warehouses ADD COLUMN IF NOT EXISTS is_default BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_warehouses_one_default_per_company
    ON company.warehouses (company_id) WHERE is_default;
