-- Trazabilidad de asientos a documentos fuente de otros módulos.
-- source_document_id   : UUID del documento (FE, DS, NA, NC, ND, nómina, etc.)
-- source_document_type : discriminador del tipo (ver constantes en journals/source.go)
-- El campo es nullable: los asientos manuales, de cierre, apertura y depreciación
-- no tienen documento fuente.

ALTER TABLE accounting.journal_entries
    ADD COLUMN IF NOT EXISTS source_document_id   UUID,
    ADD COLUMN IF NOT EXISTS source_document_type VARCHAR(30);

-- Índice para la búsqueda inversa: dado un documento, encontrar sus asientos.
CREATE INDEX IF NOT EXISTS journal_entries_source_doc_idx
    ON accounting.journal_entries (company_id, source_document_id, source_document_type)
    WHERE source_document_id IS NOT NULL;

-- En activos fijos: trazar el DS/FE de compra que originó el activo.
ALTER TABLE accounting.fixed_assets
    ADD COLUMN IF NOT EXISTS source_document_id   UUID,
    ADD COLUMN IF NOT EXISTS source_document_type VARCHAR(30);

CREATE INDEX IF NOT EXISTS fixed_assets_source_doc_idx
    ON accounting.fixed_assets (company_id, source_document_id)
    WHERE source_document_id IS NOT NULL;
