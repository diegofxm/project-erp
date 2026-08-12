-- Unidad de medida por línea (catálogo DIAN, mismo código que usa electronic.documents) -- antes
-- sales/ no la guardaba en absoluto y siempre se derivaba del producto solo al generar la
-- factura electrónica, dejando el formulario de cotización/venta sin ese campo mientras que
-- factura electrónica sí lo pedía (inconsistencia real reportada 2026-08-11).

ALTER TABLE sales.sale_lines  ADD COLUMN IF NOT EXISTS unit_code TEXT NOT NULL DEFAULT '94';
ALTER TABLE sales.quote_lines ADD COLUMN IF NOT EXISTS unit_code TEXT NOT NULL DEFAULT '94';
