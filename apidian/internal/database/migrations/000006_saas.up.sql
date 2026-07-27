-- Capa SaaS: planes, suscripciones, configuración visual, pagos, prospectos y auditoría.
-- Todo en public schema — son conceptos de la plataforma, no del dominio de documentos.

-- ── Planes ───────────────────────────────────────────────────────────────────────────────────
CREATE TABLE plans (
    id                        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                      VARCHAR(50)  NOT NULL UNIQUE,
    description               TEXT,
    max_documents_per_month   INT,                         -- NULL = ilimitado
    max_issuers               INT          NOT NULL DEFAULT 1,
    price_cop                 INT          NOT NULL DEFAULT 0,
    affiliation_fee_cop       INT          NOT NULL DEFAULT 180000,
    renewal_fee_cop           INT          NOT NULL DEFAULT 180000,
    annual_increment_pct      NUMERIC(5,2) NOT NULL DEFAULT 0,
    is_active                 BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Suscripciones ────────────────────────────────────────────────────────────────────────────
CREATE TABLE subscriptions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id   UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    plan_id     UUID        NOT NULL REFERENCES plans(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'cancelled', 'suspended')),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_subscriptions_active_issuer
    ON subscriptions(issuer_id)
    WHERE status = 'active';

-- ── Configuración y tarifas pactadas por emisor ──────────────────────────────────────────────
CREATE TABLE issuer_settings (
    issuer_id          UUID        PRIMARY KEY REFERENCES issuers(id) ON DELETE CASCADE,
    brand_color        VARCHAR(7)  NOT NULL DEFAULT '#14345C',
    affiliation_fee_cop INT        NOT NULL DEFAULT 180000,
    renewal_fee_cop     INT        NOT NULL DEFAULT 180000,
    affiliated_at       TIMESTAMPTZ,
    renewal_due_at      TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Pagos ────────────────────────────────────────────────────────────────────────────────────
CREATE TABLE payments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id  UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL CHECK (type IN ('affiliation', 'renewal', 'documents')),
    amount_cop INT         NOT NULL,
    note       TEXT,
    paid_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX payments_issuer_id_idx ON payments(issuer_id);
CREATE INDEX payments_paid_at_idx   ON payments(paid_at DESC);

-- ── Prospectos (pipeline de ventas) ──────────────────────────────────────────────────────────
CREATE TABLE prospects (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    nit         TEXT,
    cedula_pdf  BYTEA,
    rut_pdf     BYTEA,
    status      TEXT        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
    notes       TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX prospects_status_idx     ON prospects(status);
CREATE INDEX prospects_created_at_idx ON prospects(created_at DESC);

-- ── Auditoría ────────────────────────────────────────────────────────────────────────────────
CREATE TABLE audit_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id     UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    user_id       UUID        REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT        NOT NULL,
    resource_type TEXT,
    resource_id   UUID,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_events_issuer_created_idx ON audit_events(issuer_id, created_at DESC);
CREATE INDEX audit_events_resource_id_idx    ON audit_events(resource_id) WHERE resource_id IS NOT NULL;
