-- Schema de catálogos DIAN/DANE compartidos por todos los módulos.
-- Ningún otro schema tiene FK hacia public.* o edocuments.* — solo hacia catalogs.*.
-- Los datos se cargan por apidian/internal/catalogs (Seed), no aquí.
--
-- Rename respecto al esquema anterior:
--   tax_types → dian_tax_types  (elimina ambigüedad con accounting/tax/)

CREATE SCHEMA IF NOT EXISTS catalogs;

CREATE TABLE catalogs.currencies (
    code       VARCHAR(3)  PRIMARY KEY,
    name       TEXT        NOT NULL,
    symbol     VARCHAR(5)  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.departments (
    code        VARCHAR(2) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.municipalities (
    code            VARCHAR(5) PRIMARY KEY,
    name            TEXT       NOT NULL,
    department_code VARCHAR(2) NOT NULL REFERENCES catalogs.departments(code),
    description     TEXT       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.identification_types (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.payment_methods (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.payment_terms (
    code        VARCHAR(1) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Antes se llamaba tax_types en public. Renombrada para eliminar ambigüedad con
-- el módulo accounting/tax/ que calcula declaraciones de impuestos.
-- Esta tabla es el catálogo normativo DIAN de tipos de impuesto en el XML (01=IVA, 04=ICA…),
-- no tiene ninguna relación con la lógica contable de declaraciones.
CREATE TABLE catalogs.dian_tax_types (
    code        VARCHAR(2) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.tax_regimes (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.liability_codes (
    code        VARCHAR(10) PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.unit_measures (
    code        VARCHAR(3) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.dian_document_types (
    code        VARCHAR(2) PRIMARY KEY,
    name        TEXT       NOT NULL,
    description TEXT       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalogs.ciiu_codes (
    code        VARCHAR(4) PRIMARY KEY,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabla 13.3.5 del Anexo Técnico DIAN: estándares de clasificación de ítems.
-- Exactamente 4 filas fijas (UNSPSC/GTIN/Partida Arancelaria/propio).
-- agency_id = '' en la fila 999 significa que @schemeAgencyID no debe escribirse en el XML.
CREATE TABLE catalogs.item_standards (
    code        VARCHAR(10) PRIMARY KEY,
    name        TEXT        NOT NULL,
    agency_id   VARCHAR(10) NOT NULL DEFAULT '',
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
