-- Clientes: esquema unificado.
--
-- Consolida lo que antes eran 000001_customer/000002_credit_limit en una sola migración — sin
-- datos reales todavía.

CREATE SCHEMA IF NOT EXISTS customer;

CREATE TABLE IF NOT EXISTS customer.customers (
    id                      UUID PRIMARY KEY,
    company_id              UUID NOT NULL,

    -- Identificación fiscal
    identification_type_code TEXT NOT NULL DEFAULT '13',
    identification_number    TEXT NOT NULL,
    check_digit              TEXT NOT NULL DEFAULT '',
    entity_type_code         TEXT NOT NULL DEFAULT '1', -- 1=jurídica, 2=natural (DIAN cac:Party)
    merchant_registration_number TEXT NOT NULL DEFAULT '',

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
    address_city_name       TEXT NOT NULL DEFAULT '',
    address_state_name      TEXT NOT NULL DEFAULT '',
    address_country_code    TEXT NOT NULL DEFAULT 'CO',
    address_country_name    TEXT NOT NULL DEFAULT 'Colombia',

    -- Contacto
    email                   TEXT NOT NULL DEFAULT '',
    phone                   TEXT NOT NULL DEFAULT '',

    -- Cupo de crédito — NULL significa sin límite. Se valida al confirmar una venta
    -- (sales/application/confirm.go) contra la cartera pendiente del cliente.
    credit_limit             NUMERIC,

    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, identification_type_code, identification_number)
);

CREATE INDEX IF NOT EXISTS idx_customers_company ON customer.customers(company_id);
