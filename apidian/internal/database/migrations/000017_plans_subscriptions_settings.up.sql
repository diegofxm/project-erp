-- 000017: planes SaaS, suscripciones, configuración visual por emisor, y superadmin.
--
-- plans: catálogo de planes del servicio — creados y gestionados solo por superadmins.
--   max_documents_per_month NULL = ilimitado (planes premium/enterprise).
--   price_cop en pesos colombianos enteros (sin centavos) — cofacture opera solo en COP.
--
-- subscriptions: qué plan tiene cada emisor en un momento dado.
--   Solo puede haber UNA suscripción activa por emisor a la vez (uq_subscriptions_active_issuer).
--   ends_at NULL = no vence (planes de prueba/indefinidos mientras no se monetice).
--
-- issuer_settings: configuración visual/operacional por emisor que no pertenece a issuers
--   (tabla de identidad DIAN) — evita ensanchar issuers con campos de presentación.
--   Upsert (ON CONFLICT DO UPDATE) en el servicio — nunca necesita INSERT explícito.
--   brand_color es el color de acento del PDF (#RRGGBB), default igual al hardcodeado
--   en los templates actuales (#14345C).
--
-- users.is_superadmin: flag simple para el panel de administración de cofacture. No es un
--   rol RBAC completo — un boolean es suficiente para la distinción superadmin / usuario normal.
--   Se agrega con ALTER TABLE en vez de tocar la migración original (000007) para no romper
--   la secuencia de migraciones en instalaciones existentes.

-- ── Planes ───────────────────────────────────────────────────────────────────────────────────
CREATE TABLE plans (
    id                        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name                      VARCHAR(50)  NOT NULL UNIQUE,
    description               TEXT,
    max_documents_per_month   INT,                        -- NULL = ilimitado
    max_issuers               INT          NOT NULL DEFAULT 1,
    price_cop                 INT          NOT NULL DEFAULT 0,
    is_active                 BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Suscripciones ────────────────────────────────────────────────────────────────────────────
CREATE TABLE subscriptions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id   UUID         NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    plan_id     UUID         NOT NULL REFERENCES plans(id),
    status      VARCHAR(20)  NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'cancelled', 'suspended')),
    started_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    ends_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_subscriptions_active_issuer
    ON subscriptions(issuer_id)
    WHERE status = 'active';

-- ── Configuración visual por emisor ─────────────────────────────────────────────────────────
CREATE TABLE issuer_settings (
    issuer_id    UUID         PRIMARY KEY REFERENCES issuers(id) ON DELETE CASCADE,
    brand_color  VARCHAR(7)   NOT NULL DEFAULT '#14345C',
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Superadmin en users ──────────────────────────────────────────────────────────────────────
ALTER TABLE users
    ADD COLUMN is_superadmin BOOLEAN NOT NULL DEFAULT FALSE;
