-- Descuento por línea (porcentaje 0-100), aplicado antes de impuestos. Sin esto no había forma
-- de dar un descuento comercial en una cotización/venta — hueco real señalado por el usuario.
ALTER TABLE sales.sale_lines ADD COLUMN IF NOT EXISTS discount NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE sales.quote_lines ADD COLUMN IF NOT EXISTS discount NUMERIC NOT NULL DEFAULT 0;
