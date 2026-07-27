CREATE SCHEMA IF NOT EXISTS customer;

CREATE TABLE IF NOT EXISTS customer.customers (
    id                      UUID PRIMARY KEY,
    company_id              UUID NOT NULL,

    -- Identificación fiscal
    identification_type_code TEXT NOT NULL DEFAULT '13',
    identification_number    TEXT NOT NULL,
    check_digit              TEXT NOT NULL DEFAULT '',

    -- Nombre
    name                    TEXT NOT NULL,

    -- Clasificación tributaria DIAN
    tax_scheme_code         TEXT NOT NULL DEFAULT 'ZZ',
    tax_scheme_name         TEXT NOT NULL DEFAULT 'No aplica',
    tax_regime_code         TEXT,
    liability_codes         TEXT[] NOT NULL DEFAULT '{}',

    -- Ubicación
    department_code         TEXT NOT NULL DEFAULT '',
    municipality_code       TEXT NOT NULL DEFAULT '',
    address_line            TEXT NOT NULL DEFAULT '',

    -- Contacto
    email                   TEXT NOT NULL DEFAULT '',
    phone                   TEXT NOT NULL DEFAULT '',

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, identification_type_code, identification_number)
);

CREATE INDEX IF NOT EXISTS idx_customers_company ON customer.customers(company_id);
