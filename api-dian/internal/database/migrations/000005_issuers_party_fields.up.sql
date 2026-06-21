-- Fase 2.5: campos de issuers necesarios para construir cac:AccountingSupplierParty/cac:Party
-- (ubl21dian/domain.Party) — se descubrieron faltantes al construir el primer Invoice real
-- en internal/documents, reproduciendo la factura que la DIAN ya autorizó en la Fase 1.7.
--
-- liability_codes (responsabilidades fiscales, ej. "R-99-PN") es un catálogo cuyos valores
-- oficiales completos viven en la "Caja de Herramientas" de la DIAN (ver migración 000002) —
-- aquí se guarda como texto libre por emisor, no como FK a un catálogo todavía sin esa fuente.
ALTER TABLE issuers
    ADD COLUMN entity_type_code             VARCHAR(1) NOT NULL DEFAULT '1',
    ADD COLUMN tax_scheme_code               VARCHAR(2) NOT NULL DEFAULT 'ZZ' REFERENCES tax_types(code),
    ADD COLUMN tax_scheme_name                TEXT      NOT NULL DEFAULT 'No aplica',
    ADD COLUMN liability_codes                 TEXT[]    NOT NULL DEFAULT '{}',
    ADD COLUMN merchant_registration_number     TEXT;
