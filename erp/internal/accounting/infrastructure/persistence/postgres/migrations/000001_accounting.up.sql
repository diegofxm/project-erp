-- Motor contable: esquema unificado (plan de cuentas, períodos, asientos, banca, retenciones,
-- activos fijos, presupuesto y declaraciones tributarias).
--
-- Consolida lo que antes eran 000001_core/000002_modules/000003_tax/000004_fix_defaults en una
-- sola migración — sin datos reales en producción todavía, no hay historial que preservar.

CREATE SCHEMA IF NOT EXISTS accounting;

-- ── Plan de cuentas ───────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.accounts (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(10)  NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    parent_code VARCHAR(10),
    level       SMALLINT     NOT NULL CHECK (level BETWEEN 1 AND 6),
    category    VARCHAR(50)  NOT NULL,
    is_posting  BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS accounts_code_idx       ON accounting.accounts (code);
CREATE INDEX IF NOT EXISTS accounts_category_idx   ON accounting.accounts (category);
CREATE INDEX IF NOT EXISTS accounts_is_posting_idx ON accounting.accounts (is_posting) WHERE is_posting = TRUE;

-- ── Períodos contables ────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.accounting_periods (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID        NOT NULL,
    year       SMALLINT    NOT NULL,
    month      SMALLINT    NOT NULL CHECK (month BETWEEN 1 AND 12),
    status     VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'CLOSED')),
    opened_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at  TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, year, month)
);

CREATE INDEX IF NOT EXISTS periods_company_idx ON accounting.accounting_periods (company_id);

-- ── Asientos contables ────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.journal_entries (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID        NOT NULL,
    period_id            UUID        NOT NULL REFERENCES accounting.accounting_periods(id),
    date                 DATE        NOT NULL,
    description          TEXT        NOT NULL,
    status               VARCHAR(10) NOT NULL DEFAULT 'POSTED'
                             CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
    source               VARCHAR(50) NOT NULL DEFAULT 'manual',
    entry_type           VARCHAR(20) NOT NULL DEFAULT 'MANUAL'
                             CHECK (entry_type IN ('MANUAL', 'AUTOMATIC', 'ADJUSTMENT', 'CLOSING', 'OPENING')),
    voucher_type         VARCHAR(10),
    voucher_number       VARCHAR(30),
    source_document_id   UUID,
    source_document_type VARCHAR(30),
    book                 VARCHAR(10) NOT NULL DEFAULT 'BOTH'
                             CHECK (book IN ('PCGA', 'NIIF', 'BOTH')),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS journal_entries_company_idx ON accounting.journal_entries (company_id);
CREATE INDEX IF NOT EXISTS journal_entries_date_idx    ON accounting.journal_entries (date);
CREATE INDEX IF NOT EXISTS journal_entries_status_idx  ON accounting.journal_entries (status);
CREATE INDEX IF NOT EXISTS journal_entries_voucher_number_idx
    ON accounting.journal_entries (company_id, voucher_number)
    WHERE voucher_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS journal_entries_source_doc_idx
    ON accounting.journal_entries (company_id, source_document_id, source_document_type)
    WHERE source_document_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS journal_entries_book_idx
    ON accounting.journal_entries (company_id, book)
    WHERE book != 'BOTH';

CREATE TABLE IF NOT EXISTS accounting.journal_lines (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id       UUID        NOT NULL REFERENCES accounting.journal_entries(id),
    account_id       UUID        NOT NULL REFERENCES accounting.accounts(id),
    debit            BIGINT      NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit           BIGINT      NOT NULL DEFAULT 0 CHECK (credit >= 0),
    cost_center      VARCHAR(50),
    description      TEXT,
    third_party_nit  VARCHAR(20),
    foreign_amount   BIGINT,
    foreign_currency CHAR(3),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_line_amounts CHECK (
        (debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0)
    )
);

CREATE INDEX IF NOT EXISTS journal_lines_journal_idx
    ON accounting.journal_lines (journal_id);
CREATE INDEX IF NOT EXISTS journal_lines_account_idx
    ON accounting.journal_lines (account_id);
CREATE INDEX IF NOT EXISTS journal_lines_third_party_nit_idx
    ON accounting.journal_lines (third_party_nit)
    WHERE third_party_nit IS NOT NULL;
CREATE INDEX IF NOT EXISTS journal_lines_forex_idx
    ON accounting.journal_lines (foreign_currency)
    WHERE foreign_currency IS NOT NULL;

-- ── Tipos de comprobante y contadores ────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.voucher_types (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID         NOT NULL,
    code            VARCHAR(10)  NOT NULL,
    name            VARCHAR(100) NOT NULL,
    resets_annually BOOLEAN      NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, code)
);

CREATE TABLE IF NOT EXISTS accounting.voucher_counters (
    company_id UUID        NOT NULL,
    code       VARCHAR(10) NOT NULL,
    year       INTEGER     NOT NULL,
    last_seq   INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, code, year)
);

-- ── Banca ─────────────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.bank_accounts (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID         NOT NULL,
    name         VARCHAR(100) NOT NULL,
    bank_name    VARCHAR(100) NOT NULL,
    account_no   VARCHAR(50)  NOT NULL,
    account_id   UUID         NOT NULL REFERENCES accounting.accounts(id),
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, account_no)
);

CREATE TABLE IF NOT EXISTS accounting.bank_statement_lines (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_account_id UUID        NOT NULL REFERENCES accounting.bank_accounts(id),
    date            DATE        NOT NULL,
    description     TEXT        NOT NULL,
    debit           BIGINT      NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit          BIGINT      NOT NULL DEFAULT 0 CHECK (credit >= 0),
    reference       VARCHAR(100),
    is_reconciled   BOOLEAN     NOT NULL DEFAULT FALSE,
    journal_line_id UUID        REFERENCES accounting.journal_lines(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS bank_acct_company_idx    ON accounting.bank_accounts (company_id);
CREATE INDEX IF NOT EXISTS bank_stmt_account_idx    ON accounting.bank_statement_lines (bank_account_id);
CREATE INDEX IF NOT EXISTS bank_stmt_date_idx       ON accounting.bank_statement_lines (date);
CREATE INDEX IF NOT EXISTS bank_stmt_reconciled_idx ON accounting.bank_statement_lines (is_reconciled);

-- ── Retenciones ───────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.withholding_concepts (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code               VARCHAR(10)  NOT NULL,
    name               VARCHAR(200) NOT NULL,
    type               VARCHAR(20)  NOT NULL CHECK (type IN ('RETEFUENTE', 'RETEIVA', 'RETEICA')),
    rate_bp            INTEGER      NOT NULL CHECK (rate_bp > 0),
    min_base_uvt       NUMERIC(10,2) NOT NULL DEFAULT 0,
    account_payable    VARCHAR(10)  NOT NULL,
    account_receivable VARCHAR(10)  NOT NULL,
    applicable_to      VARCHAR(10)  NOT NULL DEFAULT 'BOTH'
                           CHECK (applicable_to IN ('NATURAL', 'JURIDICA', 'BOTH')),
    is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (code, type, applicable_to)
);

CREATE TABLE IF NOT EXISTS accounting.uvt_values (
    year        INTEGER PRIMARY KEY,
    value_cents BIGINT  NOT NULL CHECK (value_cents > 0)
);

CREATE INDEX IF NOT EXISTS withholding_concepts_type_idx   ON accounting.withholding_concepts (type);
CREATE INDEX IF NOT EXISTS withholding_concepts_active_idx ON accounting.withholding_concepts (is_active)
    WHERE is_active = TRUE;

-- ── Activos fijos (PPE) ───────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.fixed_assets (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID        NOT NULL,
    code                 VARCHAR(20) NOT NULL,
    name                 VARCHAR(200) NOT NULL,
    description          TEXT,
    asset_account        VARCHAR(10) NOT NULL,
    depreciation_account VARCHAR(10) NOT NULL,
    accumulated_account  VARCHAR(10) NOT NULL,
    -- 424505/529005 no existen en el PUC real (extraído de puc.com.co) — 424810 "Otros activos"
    -- y 531040 "Pérdidas por siniestros" son los reemplazos genéricos más cercanos disponibles.
    gain_account         VARCHAR(10) NOT NULL DEFAULT '424810',
    loss_account         VARCHAR(10) NOT NULL DEFAULT '531040',
    acquisition_date     DATE        NOT NULL,
    acquisition_cost     BIGINT      NOT NULL CHECK (acquisition_cost > 0),
    salvage_value        BIGINT      NOT NULL DEFAULT 0 CHECK (salvage_value >= 0),
    useful_life_months   INTEGER     NOT NULL CHECK (useful_life_months > 0),
    depreciation_method  VARCHAR(20) NOT NULL DEFAULT 'STRAIGHT_LINE'
                             CHECK (depreciation_method IN ('STRAIGHT_LINE')),
    status               VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                             CHECK (status IN ('ACTIVE', 'DISPOSED', 'FULLY_DEPRECIATED')),
    third_party_nit      VARCHAR(20),
    source_document_id   UUID,
    source_document_type VARCHAR(30),
    -- Baja del activo (disposal) — pendiente de caso de uso todavía, pero el campo se deja listo
    -- ahora que se está unificando el esquema, para no volver a alterar esta tabla después.
    disposed_at          TIMESTAMPTZ,
    disposal_amount      BIGINT,
    disposal_journal_id  UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, code)
);

CREATE TABLE IF NOT EXISTS accounting.depreciation_runs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID        NOT NULL,
    period_id  UUID        NOT NULL,
    run_date   DATE        NOT NULL,
    status     VARCHAR(20) NOT NULL DEFAULT 'COMPLETED'
                   CHECK (status IN ('COMPLETED', 'REVERSED')),
    journal_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS depreciation_runs_period_unique_idx
    ON accounting.depreciation_runs (company_id, period_id)
    WHERE status = 'COMPLETED';

CREATE TABLE IF NOT EXISTS accounting.depreciation_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id     UUID        NOT NULL REFERENCES accounting.depreciation_runs(id),
    asset_id   UUID        NOT NULL REFERENCES accounting.fixed_assets(id),
    amount     BIGINT      NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS depreciation_entries_asset_idx ON accounting.depreciation_entries (asset_id);
CREATE INDEX IF NOT EXISTS fixed_assets_source_doc_idx
    ON accounting.fixed_assets (company_id, source_document_id)
    WHERE source_document_id IS NOT NULL;

-- ── Conciliación de cartera ───────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.reconciliation_marks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID        NOT NULL,
    journal_line_id UUID        NOT NULL REFERENCES accounting.journal_lines(id),
    reconciled_with UUID        REFERENCES accounting.journal_lines(id),
    note            TEXT,
    reconciled_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (journal_line_id)
);

CREATE INDEX IF NOT EXISTS reconciliation_marks_company_idx ON accounting.reconciliation_marks (company_id);

-- ── Tasas de cambio (TRM) ─────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.exchange_rates (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rate_date     DATE        NOT NULL,
    from_currency CHAR(3)     NOT NULL,
    to_currency   CHAR(3)     NOT NULL DEFAULT 'COP',
    rate_x10000   BIGINT      NOT NULL CHECK (rate_x10000 > 0),
    source        VARCHAR(20) NOT NULL DEFAULT 'MANUAL',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT exchange_rates_unique UNIQUE (rate_date, from_currency, to_currency)
);

-- ── Presupuesto ───────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.budgets (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL,
    year       INTEGER      NOT NULL,
    name       VARCHAR(200) NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'DRAFT'
                   CHECK (status IN ('DRAFT', 'APPROVED', 'CLOSED')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, year, name)
);

CREATE TABLE IF NOT EXISTS accounting.budget_lines (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id  UUID        NOT NULL REFERENCES accounting.budgets(id) ON DELETE CASCADE,
    account_id UUID        NOT NULL REFERENCES accounting.accounts(id),
    jan        BIGINT      NOT NULL DEFAULT 0,
    feb        BIGINT      NOT NULL DEFAULT 0,
    mar        BIGINT      NOT NULL DEFAULT 0,
    apr        BIGINT      NOT NULL DEFAULT 0,
    may        BIGINT      NOT NULL DEFAULT 0,
    jun        BIGINT      NOT NULL DEFAULT 0,
    jul        BIGINT      NOT NULL DEFAULT 0,
    aug        BIGINT      NOT NULL DEFAULT 0,
    sep        BIGINT      NOT NULL DEFAULT 0,
    oct        BIGINT      NOT NULL DEFAULT 0,
    nov        BIGINT      NOT NULL DEFAULT 0,
    dec        BIGINT      NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (budget_id, account_id)
);

CREATE INDEX IF NOT EXISTS budget_lines_budget_idx ON accounting.budget_lines (budget_id);

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

-- ── Declaraciones de IVA (Formulario 300) ────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS accounting.iva_declarations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL,
    period_start     DATE        NOT NULL,
    period_end       DATE        NOT NULL,
    period_type      VARCHAR(20) NOT NULL
                         CHECK (period_type IN ('BIMESTRAL', 'CUATRIMESTRAL', 'ANUAL')),
    generated_iva    BIGINT      NOT NULL DEFAULT 0,
    deductible_iva   BIGINT      NOT NULL DEFAULT 0,
    withheld_iva     BIGINT      NOT NULL DEFAULT 0,
    net_iva          BIGINT      NOT NULL DEFAULT 0,
    previous_balance BIGINT      NOT NULL DEFAULT 0,
    amount_to_pay    BIGINT      NOT NULL DEFAULT 0,
    carry_forward    BIGINT      NOT NULL DEFAULT 0,
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                         CHECK (status IN ('DRAFT', 'FILED', 'PAID', 'CORRECTED')),
    journal_id       UUID,
    filed_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, period_start, period_end)
);

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
