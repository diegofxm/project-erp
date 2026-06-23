-- Fase 2.4: numbering_ranges — rangos de numeración autorizados por emisor y tipo de
-- documento DIAN. Una sola tabla genérica (dian_document_type_code la distingue), no una
-- tabla por tipo de documento — misma regla de naming de la sección 4.1 del architecture doc.
--
-- range_to es NULL cuando el tipo de documento no tiene un tope impuesto por una resolución
-- de la DIAN (solo Invoice exige resolución de numeración; CreditNote/DebitNote no tienen un
-- rango que se "agote" de la misma forma, pero igual necesitan un consecutivo propio que
-- nunca se repita).
--
-- current_number arranca en range_from - 1 (ver numbering.PostgresRepository.Create) para
-- que el primer ClaimNext entregue range_from con la misma fórmula que todos los siguientes.
--
-- technical_key se guarda cifrada con AES-256-GCM (internal/cryptutil, mismo mecanismo que
-- issuers) — es la clave técnica para CUFE, solo aplica a Invoice (01); CreditNote/DebitNote
-- usan el PIN del software de issuers, no esto.
CREATE TABLE numbering_ranges (
    id                       UUID         PRIMARY KEY,
    issuer_id                UUID         NOT NULL REFERENCES issuers(id),
    dian_document_type_code  VARCHAR(2)   NOT NULL REFERENCES dian_document_types(code),
    prefix                   VARCHAR(10)  NOT NULL,
    resolution_number        TEXT         NOT NULL,
    resolution_date          DATE         NOT NULL,
    range_from               BIGINT       NOT NULL,
    range_to                 BIGINT,
    current_number           BIGINT       NOT NULL,
    valid_from               DATE         NOT NULL,
    valid_to                 DATE         NOT NULL,
    environment              VARCHAR(1)   NOT NULL,
    technical_key            BYTEA,
    test_set_id              TEXT,
    is_active                BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_numbering_ranges_environment CHECK (environment IN ('1', '2')),
    CONSTRAINT chk_numbering_ranges_range CHECK (range_from > 0 AND (range_to IS NULL OR range_from <= range_to))
);

CREATE INDEX idx_numbering_ranges_issuer ON numbering_ranges(issuer_id, dian_document_type_code);
