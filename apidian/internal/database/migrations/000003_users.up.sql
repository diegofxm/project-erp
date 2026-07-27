-- Cuentas de acceso al sistema SaaS.
-- password_hash es bcrypt — nunca texto plano.
-- password_hash nullable: usuarios invitados existen antes de que completen el registro.
-- invite_token/invite_token_expires_at/invite_accepted_at: flujo de invitación de usuarios.
-- is_superadmin: panel de administración interno — boolean simple, no RBAC completo.

CREATE TABLE users (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email                    TEXT        NOT NULL UNIQUE,
    password_hash            TEXT,
    name                     TEXT        NOT NULL,
    role                     VARCHAR(20) NOT NULL DEFAULT 'admin',
    is_active                BOOLEAN     NOT NULL DEFAULT TRUE,
    is_superadmin            BOOLEAN     NOT NULL DEFAULT FALSE,
    invite_token             UUID        UNIQUE,
    invite_token_expires_at  TIMESTAMPTZ,
    invite_accepted_at       TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
