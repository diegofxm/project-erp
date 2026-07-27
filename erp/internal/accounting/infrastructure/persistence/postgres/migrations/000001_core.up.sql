-- Motor contable: núcleo.
-- Incluye el schema, el plan de cuentas, períodos, asientos contables y banca.

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
