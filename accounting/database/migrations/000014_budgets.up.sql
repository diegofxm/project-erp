CREATE TABLE accounting.budgets (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL,
    year       INT          NOT NULL,
    name       VARCHAR(200) NOT NULL,
    status     VARCHAR(20)  NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, year, name),
    CONSTRAINT budgets_status_check CHECK (status IN ('DRAFT', 'APPROVED', 'CLOSED'))
);

CREATE TABLE accounting.budget_lines (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id  UUID NOT NULL REFERENCES accounting.budgets(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounting.accounts(id),
    jan        BIGINT NOT NULL DEFAULT 0,
    feb        BIGINT NOT NULL DEFAULT 0,
    mar        BIGINT NOT NULL DEFAULT 0,
    apr        BIGINT NOT NULL DEFAULT 0,
    may        BIGINT NOT NULL DEFAULT 0,
    jun        BIGINT NOT NULL DEFAULT 0,
    jul        BIGINT NOT NULL DEFAULT 0,
    aug        BIGINT NOT NULL DEFAULT 0,
    sep        BIGINT NOT NULL DEFAULT 0,
    oct        BIGINT NOT NULL DEFAULT 0,
    nov        BIGINT NOT NULL DEFAULT 0,
    dec        BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (budget_id, account_id)
);

CREATE INDEX budget_lines_budget_idx ON accounting.budget_lines (budget_id);
