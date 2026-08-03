DROP INDEX IF EXISTS company.idx_warehouses_one_default_per_company;
ALTER TABLE company.warehouses DROP COLUMN IF EXISTS is_default;
