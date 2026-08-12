-- Unidad de medida por línea -- ver el mismo comentario en
-- sales/infrastructure/persistence/postgres/migrations/000002_sales_line_unit_code.up.sql.

ALTER TABLE purchase.order_lines ADD COLUMN IF NOT EXISTS unit_code TEXT NOT NULL DEFAULT '94';
