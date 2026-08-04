-- Productos: esquema unificado.
--
-- Consolida lo que antes eran 000001_product/000002_min_stock en una sola migración — sin datos
-- reales todavía.

CREATE SCHEMA IF NOT EXISTS product;

CREATE TABLE IF NOT EXISTS product.products (
    id                  UUID PRIMARY KEY,
    company_id          UUID NOT NULL,

    code                TEXT NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',

    -- Clasificación DIAN
    unit_measure_code        TEXT NOT NULL DEFAULT '94',  -- 94 = unidad
    standard_code             TEXT NOT NULL DEFAULT '',
    standard_code_type        TEXT NOT NULL DEFAULT 'UNSPSC', -- nombre del estándar, ej. 'UNSPSC'
    standard_code_id          TEXT NOT NULL DEFAULT '999',    -- @schemeID DIAN: 001=UNSPSC, 999=propio
    standard_code_agency_id   TEXT NOT NULL DEFAULT '',       -- @schemeAgencyID DIAN: '113' para UNSPSC

    is_service          BOOLEAN NOT NULL DEFAULT FALSE,

    -- Tributación
    tax_scheme_code     TEXT NOT NULL DEFAULT 'ZZ',
    tax_scheme_name     TEXT NOT NULL DEFAULT 'No aplica',
    tax_rate             NUMERIC(6,2) NOT NULL DEFAULT 0,

    -- Precio base
    base_price           NUMERIC(18,4) NOT NULL DEFAULT 0,

    -- Punto de reorden ("stock mínimo") — usado por el módulo inventory para resaltar
    -- existencias bajas en la pantalla de stock. 0 = sin umbral configurado.
    min_stock             NUMERIC NOT NULL DEFAULT 0,

    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, code)
);

CREATE INDEX IF NOT EXISTS idx_products_company ON product.products(company_id);
