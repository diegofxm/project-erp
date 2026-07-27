CREATE SCHEMA IF NOT EXISTS supplier;

CREATE TABLE IF NOT EXISTS supplier.suppliers (
    id                      UUID PRIMARY KEY,
    company_id              UUID NOT NULL,

    identification_type_code TEXT NOT NULL DEFAULT '31',
    identification_number    TEXT NOT NULL,
    check_digit              TEXT NOT NULL DEFAULT '',

    name                    TEXT NOT NULL,

    tax_scheme_code         TEXT NOT NULL DEFAULT 'ZZ',
    tax_scheme_name         TEXT NOT NULL DEFAULT 'No aplica',
    tax_regime_code         TEXT,
    liability_codes         TEXT[] NOT NULL DEFAULT '{}',

    department_code         TEXT NOT NULL DEFAULT '',
    municipality_code       TEXT NOT NULL DEFAULT '',
    address_line            TEXT NOT NULL DEFAULT '',

    email                   TEXT NOT NULL DEFAULT '',
    phone                   TEXT NOT NULL DEFAULT '',

    payment_terms_days      INTEGER NOT NULL DEFAULT 0,

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, identification_type_code, identification_number)
);

CREATE INDEX IF NOT EXISTS idx_suppliers_company ON supplier.suppliers(company_id);
