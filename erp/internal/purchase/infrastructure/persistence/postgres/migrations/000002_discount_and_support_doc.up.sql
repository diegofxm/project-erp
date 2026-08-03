-- Descuento por línea (porcentaje 0-100, aplicado antes de impuestos) — igual que sales.sale_lines.
ALTER TABLE purchase.order_lines ADD COLUMN IF NOT EXISTS discount NUMERIC NOT NULL DEFAULT 0;

-- Rastrea qué Documento Soporte (si alguno) se generó a partir de esta orden, para no poder
-- generar dos veces desde la misma orden. Sin FK a electronic.documents a propósito: cada
-- módulo es dueño de su schema, la integridad se maneja en la aplicación (mismo criterio que
-- sales.sales.invoice_document_id).
ALTER TABLE purchase.orders ADD COLUMN IF NOT EXISTS support_document_id UUID;
