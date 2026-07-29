CREATE SCHEMA IF NOT EXISTS company;

CREATE TABLE IF NOT EXISTS company.companies (
    id                              UUID PRIMARY KEY,
    nit                             TEXT NOT NULL UNIQUE,
    check_digit                     TEXT NOT NULL DEFAULT '',
    business_name                   TEXT NOT NULL,
    trade_name                      TEXT NOT NULL DEFAULT '',

    -- Identificación y ubicación fiscal
    identification_type_code        TEXT NOT NULL DEFAULT '31',
    department_code                 TEXT NOT NULL DEFAULT '',
    municipality_code               TEXT NOT NULL DEFAULT '',
    address_line                    TEXT NOT NULL DEFAULT '',
    email                           TEXT NOT NULL DEFAULT '',
    phone                           TEXT NOT NULL DEFAULT '',

    -- Clasificación tributaria DIAN
    environment                     VARCHAR(1) NOT NULL DEFAULT '2',
    entity_type_code                VARCHAR(2) NOT NULL DEFAULT '1',
    tax_scheme_code                 TEXT NOT NULL DEFAULT 'ZZ',
    tax_scheme_name                 TEXT NOT NULL DEFAULT 'No aplica',
    liability_codes                 TEXT[] NOT NULL DEFAULT '{}',
    tax_regime_code                 TEXT,
    industry_classification_codes   TEXT[] NOT NULL DEFAULT '{}',
    merchant_registration_number    TEXT,

    -- Credenciales DIAN — Factura Electrónica (cifradas con AES-256-GCM)
    software_id                     TEXT NOT NULL DEFAULT '',
    software_pin_enc                BYTEA,
    certificate                     BYTEA,
    certificate_password_enc        BYTEA,

    -- Credenciales DIAN — Nómina Electrónica (cifradas)
    ne_software_id                  TEXT NOT NULL DEFAULT '',
    ne_software_pin_enc             BYTEA,

    -- Visual
    logo                            BYTEA,
    logo_content_type               TEXT NOT NULL DEFAULT '',
    brand_color                     TEXT NOT NULL DEFAULT '#14345C',

    is_active                       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
