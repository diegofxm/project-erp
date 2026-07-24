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
