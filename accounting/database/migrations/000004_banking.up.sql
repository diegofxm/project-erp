-- Cuentas bancarias registradas en el sistema (una por cada cuenta del banco).
-- Cada cuenta bancaria referencia la cuenta contable del PUC que la representa
-- (p.ej. 1110 Bancos o 1120 Cuentas de ahorro).
CREATE TABLE IF NOT EXISTS accounting.bank_accounts (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID         NOT NULL,
    name         VARCHAR(100) NOT NULL,
    bank_name    VARCHAR(100) NOT NULL,
    account_no   VARCHAR(50)  NOT NULL,
    account_id   UUID         NOT NULL REFERENCES accounting.accounts (id),
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, account_no)
);

-- Líneas del extracto bancario importadas manualmente o por CSV.
-- Cada línea puede estar conciliada (matched) contra una línea de asiento contable.
CREATE TABLE IF NOT EXISTS accounting.bank_statement_lines (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    bank_account_id UUID           NOT NULL REFERENCES accounting.bank_accounts (id),
    date            DATE           NOT NULL,
    description     TEXT           NOT NULL,
    debit           NUMERIC(18, 2) NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit          NUMERIC(18, 2) NOT NULL DEFAULT 0 CHECK (credit >= 0),
    reference       VARCHAR(100),
    is_reconciled   BOOLEAN        NOT NULL DEFAULT FALSE,
    journal_line_id UUID           REFERENCES accounting.journal_lines (id),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS bank_acct_company_idx    ON accounting.bank_accounts (company_id);
CREATE INDEX IF NOT EXISTS bank_stmt_account_idx    ON accounting.bank_statement_lines (bank_account_id);
CREATE INDEX IF NOT EXISTS bank_stmt_date_idx       ON accounting.bank_statement_lines (date);
CREATE INDEX IF NOT EXISTS bank_stmt_reconciled_idx ON accounting.bank_statement_lines (is_reconciled);
