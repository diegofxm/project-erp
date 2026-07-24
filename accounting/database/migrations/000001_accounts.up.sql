CREATE SCHEMA IF NOT EXISTS accounting;

CREATE TABLE IF NOT EXISTS accounting.accounts (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(10) NOT NULL UNIQUE,
    name        VARCHAR(255) NOT NULL,
    parent_code VARCHAR(10),
    level       SMALLINT    NOT NULL CHECK (level BETWEEN 1 AND 6),
    category    VARCHAR(50) NOT NULL,
    is_posting  BOOLEAN     NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS accounts_code_idx      ON accounting.accounts (code);
CREATE INDEX IF NOT EXISTS accounts_category_idx  ON accounting.accounts (category);
CREATE INDEX IF NOT EXISTS accounts_is_posting_idx ON accounting.accounts (is_posting) WHERE is_posting = TRUE;
