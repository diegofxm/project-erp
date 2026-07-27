CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS security.users (
    id                      UUID        PRIMARY KEY,
    email                   TEXT        NOT NULL UNIQUE,
    password_hash           TEXT,
    name                    TEXT        NOT NULL,
    role                    VARCHAR(20) NOT NULL DEFAULT 'admin',
    is_active               BOOLEAN     NOT NULL DEFAULT TRUE,
    is_superadmin           BOOLEAN     NOT NULL DEFAULT FALSE,
    invite_token            UUID        UNIQUE,
    invite_token_expires_at TIMESTAMPTZ,
    invite_accepted_at      TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vínculo usuario↔empresa. FK a company schema se añade en migración futura.
CREATE TABLE IF NOT EXISTS security.user_companies (
    user_id    UUID        NOT NULL REFERENCES security.users(id) ON DELETE CASCADE,
    company_id UUID        NOT NULL,
    role       VARCHAR(20) NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, company_id)
);

CREATE INDEX IF NOT EXISTS idx_user_companies_company ON security.user_companies(company_id);
