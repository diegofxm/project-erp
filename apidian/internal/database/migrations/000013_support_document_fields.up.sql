-- Campos exclusivos del Documento Soporte (InvoiceTypeCode "05").
-- vendor: snapshot del tercero no obligado (el vendedor que no está obligado a facturar).
-- operation_type_code: "10" (Residente) o "11" (No Residente).
-- withholding_taxes: retenciones (ReteIVA/ReteRenta) que aparecen como WithholdingTaxTotal.
ALTER TABLE documents
    ADD COLUMN IF NOT EXISTS vendor              JSONB,
    ADD COLUMN IF NOT EXISTS operation_type_code VARCHAR(2),
    ADD COLUMN IF NOT EXISTS withholding_taxes   JSONB;
