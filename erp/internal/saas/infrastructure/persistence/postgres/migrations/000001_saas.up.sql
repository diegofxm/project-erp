-- SaaS: catálogo de módulos, planes, suscripciones, pagos, configuración y solicitudes de acceso.
--
-- company_id es una "FK blanda" (sin REFERENCES), mismo criterio que el resto del monolito: cada
-- módulo es dueño de su propio esquema, la integridad se garantiza en la aplicación.

CREATE SCHEMA IF NOT EXISTS saas;

-- Catálogo fijo de grupos de funcionalidad que un plan puede desbloquear (ver Sidebar del
-- frontend) — sembrado por seed, sin CRUD de superadmin (agregar un módulo real siempre implica
-- un cambio de código).
CREATE TABLE IF NOT EXISTS saas.modules (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(40)  NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas.plans (
    id                             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code                           VARCHAR(40)  NOT NULL UNIQUE,
    name                           VARCHAR(100) NOT NULL,
    description                    TEXT         NOT NULL DEFAULT '',
    billing_cycle                  VARCHAR(10)  NOT NULL
                                       CHECK (billing_cycle IN ('monthly', 'annual', 'none')),
    price_cents                    BIGINT       NOT NULL DEFAULT 0,
    included_documents             INTEGER,     -- NULL = ilimitado
    price_per_extra_document_cents BIGINT       NOT NULL DEFAULT 0,
    requires_certificate           BOOLEAN      NOT NULL DEFAULT FALSE,
    certificate_price_cents        BIGINT       NOT NULL DEFAULT 0,
    annual_increment_pct           NUMERIC(5,2) NOT NULL DEFAULT 0,
    is_internal                    BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active                      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at                     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas.plan_modules (
    plan_id   UUID NOT NULL REFERENCES saas.plans(id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES saas.modules(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, module_id)
);

CREATE TABLE IF NOT EXISTS saas.subscriptions (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id             UUID        NOT NULL,
    plan_id                UUID        NOT NULL REFERENCES saas.plans(id),
    has_own_certificate    BOOLEAN     NOT NULL DEFAULT TRUE,
    status                 VARCHAR(20) NOT NULL DEFAULT 'active'
                               CHECK (status IN ('active', 'cancelled', 'suspended')),
    contracted_price_cents BIGINT      NOT NULL DEFAULT 0,
    current_period_start   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end     TIMESTAMPTZ NOT NULL,
    cert_expires_at        TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Una sola suscripción activa por empresa a la vez.
CREATE UNIQUE INDEX IF NOT EXISTS uq_saas_subscriptions_active_company
    ON saas.subscriptions (company_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_saas_subscriptions_period_end
    ON saas.subscriptions (current_period_end) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS saas.payments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID        NOT NULL,
    subscription_id UUID,
    type            VARCHAR(20) NOT NULL CHECK (type IN ('plan', 'certificate', 'overage')),
    amount_cents    BIGINT      NOT NULL DEFAULT 0,
    note            TEXT        NOT NULL DEFAULT '',
    paid_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_saas_payments_company ON saas.payments (company_id);

-- Fila única de configuración global — tasa de IVA aplicada a todos los cobros de la plataforma.
CREATE TABLE IF NOT EXISTS saas.settings (
    id           SMALLINT    PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    iva_rate_bp  INTEGER     NOT NULL DEFAULT 1900,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS saas.prospects (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(200) NOT NULL,
    email               VARCHAR(200) NOT NULL UNIQUE,
    nit                 VARCHAR(20)  NOT NULL DEFAULT '',
    cedula_file         BYTEA,
    cedula_content_type VARCHAR(100) NOT NULL DEFAULT '',
    rut_file            BYTEA,
    rut_content_type    VARCHAR(100) NOT NULL DEFAULT '',
    status              VARCHAR(20)  NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'rejected')),
    notes               TEXT         NOT NULL DEFAULT '',
    reviewed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
