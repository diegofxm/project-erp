-- Schema edocuments: catálogos de entidades del dominio de documentos electrónicos.
-- customers, vendors y products son catálogos de conveniencia por emisor — la fuente
-- de verdad de un documento ya emitido es siempre el snapshot JSONB dentro del propio
-- documento (edocuments.documents), no estas tablas.
--
-- issuer_id UUID NOT NULL sin FK a public.issuers: el módulo edocuments es una librería
-- de dominio que no depende del schema SaaS. La API valida la existencia del emisor
-- antes de llamar a cualquier operación de este módulo.
--
-- FKs hacia catalogs.*: integridad referencial real, sin "campos volando".

CREATE SCHEMA IF NOT EXISTS edocuments;

-- ── Clientes ─────────────────────────────────────────────────────────────────────────────────
-- address_city_code/state_code sin FK: un cliente puede ser extranjero sin código DIVIPOLA.
CREATE TABLE edocuments.customers (
    id                                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id                         UUID        NOT NULL,
    entity_type_code                  VARCHAR(1),
    identification_number             TEXT        NOT NULL,
    identification_type_code          VARCHAR(3)  NOT NULL REFERENCES catalogs.identification_types(code),
    identification_verification_code  VARCHAR(1),
    name                              TEXT        NOT NULL,
    address_line                      TEXT,
    address_city_code                 TEXT,
    address_city_name                 TEXT,
    address_state_code                TEXT,
    address_state_name                TEXT,
    address_country_code              TEXT,
    address_country_name              TEXT,
    tax_scheme_code                   VARCHAR(2)  REFERENCES catalogs.dian_tax_types(code),
    tax_scheme_name                   TEXT,
    liability_codes                   TEXT[]      NOT NULL DEFAULT '{}',
    tax_regime_code                   VARCHAR(2)  REFERENCES catalogs.tax_regimes(code),
    phone                             TEXT,
    email                             TEXT,
    merchant_registration_number      TEXT,
    created_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customers_issuer ON edocuments.customers(issuer_id);

-- ── Proveedores (no obligados a facturar — DS/NA) ────────────────────────────────────────────
CREATE TABLE edocuments.vendors (
    id                                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id                         UUID        NOT NULL,
    entity_type_code                  VARCHAR(1),
    identification_number             TEXT        NOT NULL,
    identification_type_code          VARCHAR(3)  NOT NULL REFERENCES catalogs.identification_types(code),
    identification_verification_code  VARCHAR(1),
    name                              TEXT        NOT NULL,
    address_line                      TEXT,
    address_city_code                 TEXT,
    address_city_name                 TEXT,
    address_state_code                TEXT,
    address_state_name                TEXT,
    address_country_code              TEXT,
    address_country_name              TEXT,
    tax_scheme_code                   VARCHAR(2)  REFERENCES catalogs.dian_tax_types(code),
    tax_scheme_name                   TEXT,
    liability_codes                   TEXT[]      NOT NULL DEFAULT '{}',
    tax_regime_code                   VARCHAR(2)  REFERENCES catalogs.tax_regimes(code),
    phone                             TEXT,
    email                             TEXT,
    merchant_registration_number      TEXT,
    created_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vendors_issuer ON edocuments.vendors(issuer_id);

-- ── Productos / servicios ────────────────────────────────────────────────────────────────────
-- unit_measure_code ahora sí tiene FK real a catalogs.unit_measures (antes no la tenía).
-- tax_type_code referencia catalogs.dian_tax_types (el catálogo DIAN, no accounting/tax).
-- item_code único por emisor cuando no es NULL (un código puede repetirse entre emisores).
CREATE TABLE edocuments.products (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id             UUID        NOT NULL,
    description           TEXT        NOT NULL,
    unit_measure_code     VARCHAR(3)  NOT NULL REFERENCES catalogs.unit_measures(code),
    unit_price_cents      BIGINT      NOT NULL DEFAULT 0,
    item_code             TEXT,
    item_type_code        TEXT,
    item_type_name        TEXT,
    item_type_agency_id   TEXT,
    tax_type_code         VARCHAR(2)  REFERENCES catalogs.dian_tax_types(code),
    tax_type_name         TEXT,
    tax_percent           DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_issuer ON edocuments.products(issuer_id);
CREATE UNIQUE INDEX idx_products_issuer_item_code
    ON edocuments.products(issuer_id, item_code)
    WHERE item_code IS NOT NULL;
