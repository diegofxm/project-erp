-- Motor contable: declaraciones tributarias.
-- F210 Renta jurídicas, F220 Certificados de retención, F490 ICA municipal.

-- ── F210 Renta Jurídicas ──────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.income_tax_rates (
    year        INTEGER     PRIMARY KEY,
    rate_bp     INTEGER     NOT NULL CHECK (rate_bp > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounting.income_tax_declarations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL,
    fiscal_year      INTEGER     NOT NULL,
    taxable_income   BIGINT      NOT NULL DEFAULT 0,
    tax_rate_bp      INTEGER     NOT NULL,
    tax_computed     BIGINT      NOT NULL DEFAULT 0,
    discounts        BIGINT      NOT NULL DEFAULT 0,
    tax_to_pay       BIGINT      NOT NULL DEFAULT 0,
    advance_payments BIGINT      NOT NULL DEFAULT 0,
    amount_due       BIGINT      NOT NULL DEFAULT 0,
    carry_forward    BIGINT      NOT NULL DEFAULT 0,
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                         CHECK (status IN ('DRAFT', 'FILED', 'PAID', 'CORRECTED')),
    journal_id       UUID,
    filed_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT income_tax_declarations_unique UNIQUE (company_id, fiscal_year)
);

CREATE INDEX IF NOT EXISTS income_tax_declarations_company_idx
    ON accounting.income_tax_declarations (company_id);

-- ── F220 Certificados de Retención en la Fuente ───────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.withholding_certificates (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID         NOT NULL,
    fiscal_year     INTEGER      NOT NULL,
    third_party_nit VARCHAR(20)  NOT NULL,
    concept_code    VARCHAR(10)  NOT NULL,
    concept_name    VARCHAR(200) NOT NULL,
    wh_type         VARCHAR(20)  NOT NULL
                        CHECK (wh_type IN ('RETEFUENTE', 'RETEIVA', 'RETEICA')),
    gross_amount    BIGINT       NOT NULL DEFAULT 0,
    tax_withheld    BIGINT       NOT NULL DEFAULT 0,
    status          VARCHAR(20)  NOT NULL DEFAULT 'DRAFT'
                        CHECK (status IN ('DRAFT', 'ISSUED', 'CORRECTED')),
    issued_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT withholding_certificates_unique
        UNIQUE (company_id, fiscal_year, third_party_nit, concept_code, wh_type)
);

CREATE INDEX IF NOT EXISTS withholding_certificates_company_year_idx
    ON accounting.withholding_certificates (company_id, fiscal_year);
CREATE INDEX IF NOT EXISTS withholding_certificates_nit_idx
    ON accounting.withholding_certificates (third_party_nit);

-- ── F490 ICA Municipal ────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.ica_tariffs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    municipality_code VARCHAR(5)  NOT NULL,
    ciiu_code         VARCHAR(4)  NOT NULL,
    fiscal_year       INTEGER     NOT NULL,
    rate_bp           INTEGER     NOT NULL CHECK (rate_bp > 0),
    surcharge_bp      INTEGER     NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ica_tariffs_unique UNIQUE (municipality_code, ciiu_code, fiscal_year)
);

CREATE TABLE IF NOT EXISTS accounting.ica_declarations (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID        NOT NULL,
    municipality_code VARCHAR(5)  NOT NULL,
    period_start      DATE        NOT NULL,
    period_end        DATE        NOT NULL,
    period_type       VARCHAR(20) NOT NULL
                          CHECK (period_type IN ('BIMESTRAL', 'CUATRIMESTRAL', 'ANUAL')),
    ciiu_code         VARCHAR(4)  NOT NULL,
    gross_revenue     BIGINT      NOT NULL DEFAULT 0,
    deductions        BIGINT      NOT NULL DEFAULT 0,
    net_base          BIGINT      NOT NULL DEFAULT 0,
    tariff_bp         INTEGER     NOT NULL,
    surcharge_bp      INTEGER     NOT NULL DEFAULT 0,
    tax_computed      BIGINT      NOT NULL DEFAULT 0,
    surcharge_amount  BIGINT      NOT NULL DEFAULT 0,
    tax_to_pay        BIGINT      NOT NULL DEFAULT 0,
    previous_balance  BIGINT      NOT NULL DEFAULT 0,
    amount_due        BIGINT      NOT NULL DEFAULT 0,
    carry_forward     BIGINT      NOT NULL DEFAULT 0,
    status            VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                          CHECK (status IN ('DRAFT', 'FILED', 'PAID', 'CORRECTED')),
    journal_id        UUID,
    filed_at          TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ica_declarations_unique
        UNIQUE (company_id, municipality_code, period_start, period_end, ciiu_code)
);

CREATE INDEX IF NOT EXISTS ica_declarations_company_idx
    ON accounting.ica_declarations (company_id);
CREATE INDEX IF NOT EXISTS ica_declarations_municipality_idx
    ON accounting.ica_declarations (municipality_code);
