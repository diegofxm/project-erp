CREATE TABLE IF NOT EXISTS accounting.journal_entries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID        NOT NULL,
    period_id   UUID        NOT NULL REFERENCES accounting.accounting_periods (id),
    date        DATE        NOT NULL,
    description TEXT        NOT NULL,
    status      VARCHAR(10) NOT NULL DEFAULT 'POSTED' CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
    source      VARCHAR(50) NOT NULL DEFAULT 'manual',
    entry_type  VARCHAR(20) NOT NULL DEFAULT 'MANUAL'
                    CHECK (entry_type IN ('MANUAL', 'AUTOMATIC', 'ADJUSTMENT', 'CLOSING', 'OPENING')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounting.journal_lines (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id  UUID           NOT NULL REFERENCES accounting.journal_entries (id),
    account_id  UUID           NOT NULL REFERENCES accounting.accounts (id),
    debit       NUMERIC(18, 2) NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit      NUMERIC(18, 2) NOT NULL DEFAULT 0 CHECK (credit >= 0),
    cost_center VARCHAR(50),
    description TEXT,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    -- exactamente uno de debit o credit debe ser > 0
    CONSTRAINT chk_line_amounts CHECK (
        (debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0)
    )
);

CREATE INDEX IF NOT EXISTS journal_entries_company_idx ON accounting.journal_entries (company_id);
CREATE INDEX IF NOT EXISTS journal_entries_date_idx    ON accounting.journal_entries (date);
CREATE INDEX IF NOT EXISTS journal_entries_status_idx  ON accounting.journal_entries (status);
CREATE INDEX IF NOT EXISTS journal_lines_journal_idx   ON accounting.journal_lines (journal_id);
CREATE INDEX IF NOT EXISTS journal_lines_account_idx   ON accounting.journal_lines (account_id);
