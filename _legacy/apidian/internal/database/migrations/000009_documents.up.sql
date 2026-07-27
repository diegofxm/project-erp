-- Tabla central de documentos electrónicos DIAN: FE (01), NC (91), ND (92), DS (05), NA (95).
-- Una sola tabla para todos los tipos — el tipo lo discrimina dian_document_type_code.
--
-- Snapshots JSONB (customer, vendor, lines, payment_means, withholding_taxes):
--   son de solo lectura post-firma — la DIAN exige conservar el dato tal cual se firmó.
--   Borrar/editar un customer o vendor en el catálogo no afecta documentos ya emitidos.
--
-- currency_code → catalogs.currencies: FK real (antes apuntaba a public.currencies).
-- customer_id → edocuments.customers: trazabilidad opcional, ON DELETE SET NULL.
-- vendor_id   → edocuments.vendors:   trazabilidad opcional, ON DELETE SET NULL (DS/NA).
-- numbering_range_id → edocuments.numbering_ranges: rango que "gastó" el número.
--
-- prefix/number/document_key/qr_url/signed_xml nullable: un borrador (DRAFT) existe
--   antes de reclamar su consecutivo real — estos campos se llenan al confirmar.
--
-- uq_documents_range_number: índice PARCIAL — excluye rejected/send_error (números
--   no gastados definitivamente; el siguiente intento los reclama con ReleaseIfCurrent).

CREATE TABLE edocuments.documents (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id                   UUID        NOT NULL,
    numbering_range_id          UUID        NOT NULL REFERENCES edocuments.numbering_ranges(id),
    dian_document_type_code     VARCHAR(2)  NOT NULL REFERENCES catalogs.dian_document_types(code),
    currency_code               VARCHAR(3)  NOT NULL REFERENCES catalogs.currencies(code),
    prefix                      VARCHAR(10),
    number                      BIGINT,
    document_key                TEXT,
    issue_date                  DATE,
    issue_time                  TEXT,

    -- Snapshots de solo lectura — no son FK, son la verdad firmada y sellada
    customer                    JSONB       NOT NULL,
    customer_id                 UUID        REFERENCES edocuments.customers(id) ON DELETE SET NULL,
    vendor                      JSONB,
    vendor_id                   UUID        REFERENCES edocuments.vendors(id)   ON DELETE SET NULL,
    operation_type_code         VARCHAR(2),
    lines                       JSONB       NOT NULL,
    payment_means               JSONB       NOT NULL DEFAULT '[]',
    withholding_taxes           JSONB,

    -- Totales en centavos (BIGINT, no float)
    totals_line_extension_cents BIGINT      NOT NULL,
    totals_tax_exclusive_cents  BIGINT      NOT NULL,
    totals_tax_inclusive_cents  BIGINT      NOT NULL,
    totals_prepaid_cents        BIGINT      NOT NULL DEFAULT 0,
    totals_payable_cents        BIGINT      NOT NULL,

    -- Solo NC/ND/NA
    billing_reference           JSONB,
    discrepancy_response        JSONB,
    note_type_code              VARCHAR(2),
    note                        TEXT,

    -- Resultado DIAN (se llena después de enviar)
    qr_url                      TEXT,
    signed_xml                  TEXT,
    status                      VARCHAR(20) NOT NULL,
    dian_track_id               TEXT,
    dian_status_code            TEXT,
    dian_status_description     TEXT,
    dian_status_message         TEXT,
    application_response_xml    TEXT,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_issuer
    ON edocuments.documents(issuer_id, dian_document_type_code);
CREATE INDEX idx_documents_document_key
    ON edocuments.documents(document_key);
CREATE INDEX idx_documents_customer
    ON edocuments.documents(customer_id) WHERE customer_id IS NOT NULL;
CREATE INDEX idx_documents_vendor
    ON edocuments.documents(vendor_id)   WHERE vendor_id   IS NOT NULL;
CREATE UNIQUE INDEX uq_documents_range_number
    ON edocuments.documents(numbering_range_id, number)
    WHERE status NOT IN ('rejected', 'send_error');
