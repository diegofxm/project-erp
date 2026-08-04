-- Terceros: reemplaza los antiguos schemas customer/ y supplier/ (dos tablas casi idénticas
-- mantenidas por separado) por una sola tabla con roles independientes — un mismo tercero puede
-- ser cliente y proveedor de la misma empresa a la vez, sin duplicar su identificación fiscal.

CREATE SCHEMA IF NOT EXISTS thirdparty;

CREATE TABLE IF NOT EXISTS thirdparty.parties (
    id                      UUID PRIMARY KEY,
    company_id              UUID NOT NULL,

    identification_type_code     TEXT NOT NULL DEFAULT '13',
    identification_number        TEXT NOT NULL,
    check_digit                  TEXT NOT NULL DEFAULT '',
    entity_type_code             TEXT NOT NULL DEFAULT '1', -- 1=jurídica, 2=natural (DIAN cac:Party)
    merchant_registration_number TEXT NOT NULL DEFAULT '',

    name                    TEXT NOT NULL,

    tax_scheme_code         TEXT NOT NULL DEFAULT 'ZZ',
    tax_scheme_name         TEXT NOT NULL DEFAULT 'No aplica',
    tax_regime_code         TEXT,
    liability_codes         TEXT[] NOT NULL DEFAULT '{}',

    department_code         TEXT NOT NULL DEFAULT '',
    municipality_code       TEXT NOT NULL DEFAULT '',
    address_line            TEXT NOT NULL DEFAULT '',
    address_city_name       TEXT NOT NULL DEFAULT '',
    address_state_name      TEXT NOT NULL DEFAULT '',
    address_country_code    TEXT NOT NULL DEFAULT 'CO',
    address_country_name    TEXT NOT NULL DEFAULT 'Colombia',

    email                   TEXT NOT NULL DEFAULT '',
    phone                   TEXT NOT NULL DEFAULT '',

    -- Roles independientes — un mismo tercero puede ser cliente y proveedor a la vez.
    is_customer              BOOLEAN NOT NULL DEFAULT FALSE,
    is_supplier              BOOLEAN NOT NULL DEFAULT FALSE,

    -- Campos específicos de rol — conviven en la misma fila.
    credit_limit             NUMERIC,                     -- solo aplica si is_customer
    payment_terms_days       INTEGER NOT NULL DEFAULT 0,   -- solo aplica si is_supplier

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, identification_type_code, identification_number)
);

CREATE INDEX IF NOT EXISTS idx_parties_company          ON thirdparty.parties(company_id);
CREATE INDEX IF NOT EXISTS idx_parties_company_customer  ON thirdparty.parties(company_id) WHERE is_customer;
CREATE INDEX IF NOT EXISTS idx_parties_company_supplier  ON thirdparty.parties(company_id) WHERE is_supplier;
