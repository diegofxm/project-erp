-- Fase 2.9 (internal/auth): users — la cuenta de acceso de un emisor/tenant. "Un usuario = un
-- emisor": issuer_id es obligatorio y fijo (decisión explícita del usuario, ver
-- docs/api-dian-architecture.md sección 9.17) — no hay tabla intermedia user_issuers todavía.
--
-- password_hash es bcrypt (internal/auth/password.go), nunca texto plano ni cifrado reversible
-- (a diferencia de issuers.software_pin/certificate, que SÍ se descifran para usarse con
-- ubl21dian — una contraseña de login nunca necesita recuperarse, solo verificarse).
CREATE TABLE users (
    id              UUID         PRIMARY KEY,
    issuer_id       UUID         NOT NULL REFERENCES issuers(id),
    email           TEXT         NOT NULL UNIQUE,
    password_hash   TEXT         NOT NULL,
    name            TEXT         NOT NULL,
    role            VARCHAR(20)  NOT NULL DEFAULT 'admin',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_issuer ON users(issuer_id);
