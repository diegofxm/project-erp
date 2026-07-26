-- ── F210 Renta Jurídicas ──────────────────────────────────────────────────────────────────────
--
-- income_tax_rates: tasas históricas de renta (ley colombiana). Permiten cambiar la tasa anual
-- sin tocar código. Valores sembrados en la migración porque son datos normativos, no contenido.
CREATE TABLE IF NOT EXISTS accounting.income_tax_rates (
    year        INTEGER     PRIMARY KEY,
    rate_bp     INTEGER     NOT NULL CHECK (rate_bp > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tasas históricas (Ley 1943/2019, Ley 2010/2019, Ley 2155/2021):
-- 2019: 33%, 2020: 32%, 2021: 31%, 2022+: 35% (reforma de 2021 revirtió reducción gradual).
INSERT INTO accounting.income_tax_rates (year, rate_bp) VALUES
    (2019, 3300),
    (2020, 3200),
    (2021, 3100),
    (2022, 3500),
    (2023, 3500),
    (2024, 3500),
    (2025, 3500),
    (2026, 3500)
ON CONFLICT (year) DO NOTHING;

CREATE TABLE IF NOT EXISTS accounting.income_tax_declarations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL,
    fiscal_year      INTEGER     NOT NULL,
    taxable_income   BIGINT      NOT NULL DEFAULT 0,  -- renta líquida gravable en centavos
    tax_rate_bp      INTEGER     NOT NULL,             -- snapshot de la tasa al momento del cálculo
    tax_computed     BIGINT      NOT NULL DEFAULT 0,  -- taxable_income * tax_rate_bp / 10000
    discounts        BIGINT      NOT NULL DEFAULT 0,  -- descuentos tributarios
    tax_to_pay       BIGINT      NOT NULL DEFAULT 0,  -- tax_computed − discounts
    advance_payments BIGINT      NOT NULL DEFAULT 0,  -- anticipos + retenciones a favor (135505)
    amount_due       BIGINT      NOT NULL DEFAULT 0,  -- max(0, tax_to_pay − advance_payments)
    carry_forward    BIGINT      NOT NULL DEFAULT 0,  -- max(0, advance_payments − tax_to_pay)
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                         CHECK (status IN ('DRAFT','FILED','PAID','CORRECTED')),
    journal_id       UUID,                            -- asiento de pago (NULL hasta PAID)
    filed_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT income_tax_declarations_unique UNIQUE (company_id, fiscal_year)
);

CREATE INDEX IF NOT EXISTS income_tax_declarations_company_idx
    ON accounting.income_tax_declarations (company_id);

-- ── F220 Certificados de Retención en la Fuente ───────────────────────────────────────────────
--
-- Almacena el certificado por empresa × año × NIT tercero × concepto × tipo de retención.
-- Los montos se calculan desde journal_lines (account_code = withholding_concepts.account_payable).
-- concept_code se guarda como el account_code de la cuenta de retención para poder recalcular.
-- No hay FK a withholding_concepts: misma tabla del mismo módulo, se une por código en la query.
CREATE TABLE IF NOT EXISTS accounting.withholding_certificates (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID         NOT NULL,
    fiscal_year     INTEGER      NOT NULL,
    third_party_nit VARCHAR(20)  NOT NULL,
    concept_code    VARCHAR(10)  NOT NULL,   -- account_code de la cuenta de retención
    concept_name    VARCHAR(200) NOT NULL,
    wh_type         VARCHAR(20)  NOT NULL
                        CHECK (wh_type IN ('RETEFUENTE','RETEIVA','RETEICA')),
    gross_amount    BIGINT       NOT NULL DEFAULT 0,  -- base gravable estimada (tax_withheld × 10000/rate)
    tax_withheld    BIGINT       NOT NULL DEFAULT 0,  -- valor retenido (suma créditos del período)
    status          VARCHAR(20)  NOT NULL DEFAULT 'DRAFT'
                        CHECK (status IN ('DRAFT','ISSUED','CORRECTED')),
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

-- ── F490 ICA por Municipio ────────────────────────────────────────────────────────────────────
--
-- ica_tariffs: tarifa por municipio + CIIU + año.
-- municipality_code y ciiu_code se almacenan sin FK a public.municipalities / public.ciiu_codes
-- para respetar el aislamiento de módulos (mismo patrón que exchange_rates.from_currency).
CREATE TABLE IF NOT EXISTS accounting.ica_tariffs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    municipality_code VARCHAR(5)  NOT NULL,   -- código DANE; sin FK (aislamiento de módulos)
    ciiu_code         VARCHAR(4)  NOT NULL,   -- código CIIU; sin FK (aislamiento de módulos)
    fiscal_year       INTEGER     NOT NULL,
    rate_bp           INTEGER     NOT NULL CHECK (rate_bp > 0),   -- tarifa en basis points (1000 = 10‰)
    surcharge_bp      INTEGER     NOT NULL DEFAULT 0,             -- sobretasa (0 si no aplica)
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
                          CHECK (period_type IN ('BIMESTRAL','CUATRIMESTRAL','ANUAL')),
    ciiu_code         VARCHAR(4)  NOT NULL,
    gross_revenue     BIGINT      NOT NULL DEFAULT 0,
    deductions        BIGINT      NOT NULL DEFAULT 0,
    net_base          BIGINT      NOT NULL DEFAULT 0,  -- gross_revenue − deductions
    tariff_bp         INTEGER     NOT NULL,             -- snapshot de la tarifa al calcular
    surcharge_bp      INTEGER     NOT NULL DEFAULT 0,
    tax_computed      BIGINT      NOT NULL DEFAULT 0,  -- net_base * tariff_bp / 10000
    surcharge_amount  BIGINT      NOT NULL DEFAULT 0,  -- net_base * surcharge_bp / 10000
    tax_to_pay        BIGINT      NOT NULL DEFAULT 0,  -- tax_computed + surcharge_amount
    previous_balance  BIGINT      NOT NULL DEFAULT 0,  -- saldo a favor del período anterior
    amount_due        BIGINT      NOT NULL DEFAULT 0,  -- max(0, tax_to_pay − previous_balance)
    carry_forward     BIGINT      NOT NULL DEFAULT 0,  -- max(0, previous_balance − tax_to_pay)
    status            VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                          CHECK (status IN ('DRAFT','FILED','PAID','CORRECTED')),
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
