-- Fase 2.5/2.6: documents — UNA tabla para todos los tipos de documento DIAN (Invoice,
-- CreditNote, DebitNote), no una tabla por tipo — misma regla de naming de la sección 4.1 del
-- architecture doc. billing_reference/discrepancy_response son NULL en Invoice, solo aplican
-- a las notas.
--
-- customer/lines/payment_means son snapshots JSONB de solo lectura (pass-through, sin tablas
-- customers/products propias) — se reciben en el payload de emisión y se persisten tal cual
-- porque eso es lo que la ley exige conservar, no porque api-dian los gestione como entidades.
--
-- signed_xml guarda el XML firmado completo — retención legal, nunca se recalcula después de
-- firmado (mismo principio que ubl21dian: nunca mutar un documento ya firmado).
CREATE TABLE documents (
    id                       UUID         PRIMARY KEY,
    issuer_id                UUID         NOT NULL REFERENCES issuers(id),
    numbering_range_id       UUID         NOT NULL REFERENCES numbering_ranges(id),
    dian_document_type_code  VARCHAR(2)   NOT NULL REFERENCES dian_document_types(code),
    prefix                   VARCHAR(10)  NOT NULL,
    number                   BIGINT       NOT NULL,
    document_key             TEXT         NOT NULL,
    issue_date               DATE         NOT NULL,
    issue_time               TEXT         NOT NULL,
    currency_code            VARCHAR(3)   NOT NULL REFERENCES currencies(code),

    customer                 JSONB        NOT NULL,
    lines                    JSONB        NOT NULL,
    payment_means            JSONB        NOT NULL DEFAULT '[]',

    totals_line_extension_cents BIGINT    NOT NULL,
    totals_tax_exclusive_cents  BIGINT    NOT NULL,
    totals_tax_inclusive_cents  BIGINT    NOT NULL,
    totals_prepaid_cents        BIGINT    NOT NULL DEFAULT 0,
    totals_payable_cents         BIGINT   NOT NULL,

    billing_reference         JSONB, -- solo CreditNote/DebitNote
    discrepancy_response      JSONB, -- solo CreditNote/DebitNote, opcional incluso ahí
    note_type_code            VARCHAR(2), -- CreditNoteTypeCode — solo CreditNote lo tiene en ubl21dian

    qr_url                    TEXT        NOT NULL,
    signed_xml                 TEXT       NOT NULL,

    status                      VARCHAR(20) NOT NULL,
    dian_track_id                TEXT,
    dian_status_code              TEXT,
    dian_status_description         TEXT, -- texto humano de dian.Result, ej. "Set de prueba ... Aceptado"
    dian_status_message              TEXT, -- distinto de dian_status_description: la DIAN los usa para cosas distintas

    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_documents_range_number UNIQUE (numbering_range_id, number)
);

CREATE INDEX idx_documents_issuer ON documents(issuer_id, dian_document_type_code);
CREATE INDEX idx_documents_document_key ON documents(document_key);
