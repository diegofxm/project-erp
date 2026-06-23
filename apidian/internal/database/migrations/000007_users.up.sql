-- Fase 2.9 (internal/auth): users — la cuenta de acceso. Originalmente "un usuario = un
-- emisor" con issuer_id NOT NULL aquí mismo — reemplazado en la Fase 9.32 por la tabla
-- user_issuers (000009): un usuario puede no tener todavía ninguna empresa (registro
-- desacoplado) o tener varias (multi-empresa/sucursales), ver
-- docs/apidian-architecture.md sección 9.32.
--
-- password_hash es bcrypt (internal/auth/password.go), nunca texto plano ni cifrado reversible
-- (a diferencia de issuers.software_pin/certificate, que SÍ se descifran para usarse con
-- cofacture — una contraseña de login nunca necesita recuperarse, solo verificarse).
CREATE TABLE users (
    id              UUID         PRIMARY KEY,
    email           TEXT         NOT NULL UNIQUE,
    password_hash   TEXT         NOT NULL,
    name            TEXT         NOT NULL,
    role            VARCHAR(20)  NOT NULL DEFAULT 'admin',
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
