-- Schema de catálogos DIAN/DANE compartidos por todos los módulos.
-- Usa IF NOT EXISTS en todas las sentencias para ser idempotente con instalaciones
-- donde apidian ya haya creado el schema vía su propia migración.

CREATE SCHEMA IF NOT EXISTS catalogs;

CREATE TABLE IF NOT EXISTS catalogs.currencies (
    code       VARCHAR(3)  PRIMARY KEY,
    name       TEXT        NOT NULL,
    symbol     VARCHAR(5)  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.departments (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.municipalities (
    code            VARCHAR(5) PRIMARY KEY,
    name            TEXT       NOT NULL,
    department_code VARCHAR(2) NOT NULL REFERENCES catalogs.departments(code),
    description     TEXT       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.identification_types (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.payment_methods (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.payment_terms (
    code        VARCHAR(1) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.dian_tax_types (
    code        VARCHAR(2) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.tax_regimes (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.liability_codes (
    code        VARCHAR(10) PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.unit_measures (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.dian_document_types (
    code        VARCHAR(2) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.ciiu_codes (
    code        VARCHAR(4) PRIMARY KEY,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalogs.item_standards (
    code        VARCHAR(10) PRIMARY KEY,
    name        TEXT        NOT NULL,
    agency_id   VARCHAR(10) NOT NULL DEFAULT '',
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
