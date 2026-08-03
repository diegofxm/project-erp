-- Rastrea qué factura electrónica (si alguna) se generó a partir de esta venta, para no poder
-- generar dos veces desde la misma venta (ver electronic/application CreateFromSaleUseCase).
-- Sin FK a electronic.documents a propósito: cada módulo es dueño de su schema, la integridad
-- se maneja en la aplicación, no en la base de datos (mismo criterio que product_id en
-- sale_lines, sin referencia cruzada a product.products).
ALTER TABLE sales.sales ADD COLUMN IF NOT EXISTS invoice_document_id UUID;
