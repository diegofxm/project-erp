-- Fase 2.5/2.6: documents — UNA tabla para todos los tipos de documento DIAN (Invoice,
-- CreditNote, DebitNote), no una tabla por tipo — misma regla de naming de la sección 4.1 del
-- architecture doc. billing_reference/discrepancy_response son NULL en Invoice, solo aplican
-- a las notas.
--
-- customer/lines/payment_means son snapshots JSONB de solo lectura (pass-through, sin tablas
-- customers/products propias) — se reciben en el payload de emisión y se persisten tal cual
-- porque eso es lo que la ley exige conservar, no porque apidian los gestione como entidades.
--
-- signed_xml guarda el XML firmado completo — retención legal, nunca se recalcula después de
-- firmado (mismo principio que cofacture: nunca mutar un documento ya firmado).
--
-- prefix/number/document_key/issue_date/issue_time/qr_url/signed_xml son NULLABLE porque un
-- documento puede existir como "draft" (status, ver internal/documents/model.go) antes de
-- reclamar su consecutivo real — no hay número/CUFE/firma todavía. Solo se llenan al
-- confirmar (POST /documents/{id}/confirm), el único punto donde se "gasta" un número real de
-- la DIAN.
--
-- uq_documents_range_number es un índice PARCIAL (Fase 9.33), no una constraint plana: excluye
-- status 'rejected'/'send_error' a propósito. Un documento rechazado por la DIAN nunca quedó
-- realmente registrado (no es como un documento ACEPTADO que se anula con una Nota Crédito,
-- ese sí queda gastado para siempre); y un send_error significa que ni siquiera se intentó
-- transmitir. numbering.Service.ReleaseIfCurrent reclama ese mismo número para el siguiente
-- intento en vez de avanzar — sin este índice parcial, el segundo documento con el mismo
-- número chocaría contra el primero (rechazado) que sigue ocupando la fila. Postgres nunca
-- considera dos NULL iguales entre sí en un índice único, así que varios borradores sin número
-- (number NULL) tampoco chocan entre ellos — eso sigue igual.
--
-- customer_id: referencia OPCIONAL y de solo trazabilidad al catálogo de clientes — NUNCA la
-- fuente de verdad del documento (eso sigue siendo la columna "customer", el snapshot JSONB
-- que de verdad se firma y se envía a la DIAN), ni se serializa en ningún XML — no afecta el
-- cumplimiento del Anexo Técnico. Esta migración va DESPUÉS de 000005_customers a propósito
-- (renumerada el 2026-06-23, sin datos reales en juego): así customer_id se crea inline aquí
-- mismo, en su posición lógica, en vez de un ALTER TABLE posterior (que la habría dejado
-- físicamente después de created_at/updated_at, violando la convención de esta sección — ver
-- docs/apidian-architecture.md sección 9.31). Nullable (una factura con datos sueltos, sin
-- cliente guardado, no la tiene) y ON DELETE SET NULL (borrar un cliente nunca debe romper ni
-- borrar documentos ya emitidos).
CREATE TABLE documents (
    id                       UUID         PRIMARY KEY,
    issuer_id                UUID         NOT NULL REFERENCES issuers(id),
    numbering_range_id       UUID         NOT NULL REFERENCES numbering_ranges(id),
    dian_document_type_code  VARCHAR(2)   NOT NULL REFERENCES dian_document_types(code),
    prefix                   VARCHAR(10),
    number                   BIGINT,
    document_key             TEXT,
    issue_date               DATE,
    issue_time               TEXT,
    currency_code            VARCHAR(3)   NOT NULL REFERENCES currencies(code),

    customer                 JSONB        NOT NULL,
    customer_id              UUID         REFERENCES customers(id) ON DELETE SET NULL,
    lines                    JSONB        NOT NULL,
    payment_means            JSONB        NOT NULL DEFAULT '[]',

    totals_line_extension_cents BIGINT    NOT NULL,
    totals_tax_exclusive_cents  BIGINT    NOT NULL,
    totals_tax_inclusive_cents  BIGINT    NOT NULL,
    totals_prepaid_cents        BIGINT    NOT NULL DEFAULT 0,
    totals_payable_cents         BIGINT   NOT NULL,

    billing_reference         JSONB, -- solo CreditNote/DebitNote
    discrepancy_response      JSONB, -- solo CreditNote/DebitNote, opcional incluso ahí
    note_type_code            VARCHAR(2), -- CreditNoteTypeCode — solo CreditNote lo tiene en cofacture
    note                      TEXT, -- cbc:Note del XML — se guarda desde el borrador, no solo al confirmar

    qr_url                    TEXT,
    signed_xml                 TEXT,

    status                      VARCHAR(20) NOT NULL,
    dian_track_id                TEXT,
    dian_status_code              TEXT,
    dian_status_description         TEXT, -- texto humano de dian.Result, ej. "Set de prueba ... Aceptado"
    dian_status_message              TEXT, -- distinto de dian_status_description: la DIAN los usa para cosas distintas
    application_response_xml         TEXT, -- XML del ApplicationResponse de la DIAN; se persiste para construir el AttachedDocument UBL al enviar por correo (ver email.go y sección 9.49 del architecture doc)

    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_issuer ON documents(issuer_id, dian_document_type_code);
CREATE INDEX idx_documents_document_key ON documents(document_key);
CREATE INDEX idx_documents_customer ON documents(customer_id) WHERE customer_id IS NOT NULL;
CREATE UNIQUE INDEX uq_documents_range_number ON documents(numbering_range_id, number)
    WHERE status NOT IN ('rejected', 'send_error');
