-- Referencia de trazabilidad al catálogo de proveedores — mismo criterio que customer_id:
-- NO es la fuente de verdad del Documento Soporte (eso sigue siendo documents.vendor JSONB),
-- solo sirve para que VendorSection pueda mostrar el proveedor guardado al reabrir un borrador.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS vendor_id UUID REFERENCES vendors(id);
