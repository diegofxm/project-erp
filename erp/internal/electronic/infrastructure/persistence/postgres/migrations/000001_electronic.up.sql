CREATE SCHEMA IF NOT EXISTS electronic;

CREATE TABLE IF NOT EXISTS electronic.numbering_ranges (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID        NOT NULL,
    dian_document_type_code VARCHAR(10) NOT NULL,
    prefix                VARCHAR(20),
    resolution_number     VARCHAR(100),
    resolution_date       DATE,
    range_from            BIGINT      NOT NULL DEFAULT 1,
    range_to              BIGINT,
    current_number        BIGINT      NOT NULL DEFAULT 0,
    valid_from            DATE,
    valid_to              DATE,
    environment           VARCHAR(2)  NOT NULL DEFAULT '2',
    technical_key         BYTEA,
    test_set_id           TEXT        NOT NULL DEFAULT '',
    is_active             BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_numbering_ranges_company
    ON electronic.numbering_ranges (company_id);

CREATE TABLE IF NOT EXISTS electronic.documents (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id                  UUID        NOT NULL,
    numbering_range_id          UUID        NOT NULL REFERENCES electronic.numbering_ranges(id),
    dian_document_type_code     VARCHAR(10) NOT NULL,
    prefix                      VARCHAR(20),
    number                      BIGINT,
    document_key                TEXT,
    issue_date                  DATE,
    issue_time                  TEXT,
    currency_code               VARCHAR(10) NOT NULL DEFAULT 'COP',
    customer                    JSONB       NOT NULL DEFAULT '{}',
    customer_id                 UUID,
    lines                       JSONB       NOT NULL DEFAULT '[]',
    payment_means               JSONB       NOT NULL DEFAULT '[]',
    totals_line_extension_cents BIGINT      NOT NULL DEFAULT 0,
    totals_tax_exclusive_cents  BIGINT      NOT NULL DEFAULT 0,
    totals_tax_inclusive_cents  BIGINT      NOT NULL DEFAULT 0,
    totals_prepaid_cents        BIGINT      NOT NULL DEFAULT 0,
    totals_payable_cents        BIGINT      NOT NULL DEFAULT 0,
    billing_reference           JSONB,
    discrepancy_response        JSONB,
    note_type_code              TEXT,
    note                        TEXT,
    qr_url                      TEXT,
    signed_xml                  TEXT,
    status                      VARCHAR(20) NOT NULL DEFAULT 'draft',
    dian_track_id               TEXT,
    dian_status_code            TEXT,
    dian_status_description     TEXT,
    dian_status_message         TEXT,
    application_response_xml    TEXT,
    supplier                    JSONB,
    operation_type_code         VARCHAR(5),
    withholding_taxes           JSONB,
    supplier_id                 UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_company
    ON electronic.documents (company_id);
CREATE INDEX IF NOT EXISTS idx_documents_status
    ON electronic.documents (company_id, status);
CREATE INDEX IF NOT EXISTS idx_documents_issue_date
    ON electronic.documents (company_id, issue_date DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_document_key
    ON electronic.documents (company_id, document_key)
    WHERE document_key IS NOT NULL;
-- FKs sin índice propio (cubiertas hoy solo indirectamente por idx_documents_company) --
-- baratas de añadir ahora, antes de que reportes de rangos/terceros empiecen a filtrar por ellas.
CREATE INDEX IF NOT EXISTS idx_documents_numbering_range
    ON electronic.documents (numbering_range_id);
CREATE INDEX IF NOT EXISTS idx_documents_customer
    ON electronic.documents (customer_id) WHERE customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_documents_supplier
    ON electronic.documents (supplier_id) WHERE supplier_id IS NOT NULL;
