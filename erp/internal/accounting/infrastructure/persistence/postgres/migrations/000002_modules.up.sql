-- Motor contable: módulos operativos.
-- Retenciones, activos fijos, IVA (F300), conciliación, forex y presupuesto.

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
    gain_account         VARCHAR(10) NOT NULL DEFAULT '424505',
    loss_account         VARCHAR(10) NOT NULL DEFAULT '529005',
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
