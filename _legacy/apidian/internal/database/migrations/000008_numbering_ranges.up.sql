-- Rangos de numeración DIAN por emisor y tipo de documento.
-- Una sola tabla genérica; dian_document_type_code la distingue.
--
-- resolution_number/resolution_date/valid_from/valid_to nullable: NC (91), ND (92) y
--   NA (95) no requieren resolución DIAN — solo FE (01) y DS (05) la necesitan.
-- current_number arranca en range_from - 1 para que el primer ClaimNext entregue range_from.
-- technical_key cifrada AES-256-GCM: clave técnica para CUFE (solo aplica a FE/01).
--
-- uq_numbering_ranges_range_number: índice PARCIAL — excluye rejected/send_error porque
--   la DIAN no los considera "gastados"; el siguiente intento reclama el mismo número.

CREATE TABLE edocuments.numbering_ranges (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id                UUID        NOT NULL,
    dian_document_type_code  VARCHAR(2)  NOT NULL REFERENCES catalogs.dian_document_types(code),
    prefix                   VARCHAR(10) NOT NULL,
    resolution_number        TEXT,
    resolution_date          DATE,
    range_from               BIGINT      NOT NULL,
    range_to                 BIGINT,
    current_number           BIGINT      NOT NULL,
    valid_from               DATE,
    valid_to                 DATE,
    environment              VARCHAR(1)  NOT NULL,
    technical_key            BYTEA,
    test_set_id              TEXT,
    is_active                BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_numbering_ranges_environment CHECK (environment IN ('1', '2')),
    CONSTRAINT chk_numbering_ranges_range       CHECK (range_from > 0 AND (range_to IS NULL OR range_from <= range_to))
);

CREATE INDEX idx_numbering_ranges_issuer
    ON edocuments.numbering_ranges(issuer_id, dian_document_type_code);
