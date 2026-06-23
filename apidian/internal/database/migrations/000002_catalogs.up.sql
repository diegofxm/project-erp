-- Fase 2.2: esquema de catálogos DIAN — solo estructura. Los datos se cargan por código
-- desde internal/database/seed/*.csv (ver database.Seed), no aquí: catálogos como
-- municipalities tienen cientos de filas reales y no escalan como INSERT literal en una
-- migración, y además así se pueden actualizar sin escribir una migración nueva cada vez.
--
-- Alcance recortado al de este orquestador (Invoice 01 / CreditNote 91 / DebitNote 92, ver
-- docs/apidian-architecture.md sección 9.2): no incluye catálogos de Documento Soporte,
-- Tiquete POS, Documentos Equivalentes ni Nómina — esos no aplican todavía.
--
-- Catálogos pendientes (NO incluidos a propósito): tax_level_codes (responsabilidades
-- fiscales O-13/O-15/O-47...), credit_note_concepts, debit_note_concepts, countries,
-- type_organizations, type_regimes. El Anexo Técnico 1.9 remite esas tablas a la "Caja de
-- Herramientas Factura Electrónica" (archivo .xlsx aparte de la DIAN, secciones 13.2.7.4 y
-- 13.2.7.5) que no está en este repositorio — se agregan cuando se tenga esa fuente oficial,
-- para no inventar códigos de cumplimiento tributario.

CREATE TABLE currencies (
    code        VARCHAR(3)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    symbol      VARCHAR(5)  NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE departments (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tipos de identificación de personas — no confundir con dian_document_types (tipos de
-- documento ELECTRÓNICO: factura, notas...), son catálogos distintos que la palabra
-- "document" por sí sola no distingue.
--
-- El código es el numérico oficial de la DIAN (13 cédula, 31 NIT, etc.), NO una abreviatura
-- legible ("CC"/"NIT") — se usa directamente como cbc:CompanyID.@schemeName /
-- sts:ProviderID.@schemeName en el XML (cofacture/builder/party.go), sin ninguna traducción
-- intermedia. Primer intento de esta tabla (Fase 2.2) sí usaba abreviaturas — se corrigió en
-- la Fase 2.8 al fallar un envío real con "errores en campos mandatorios": "13" y "31" están
-- confirmados contra una factura real autorizada por la DIAN
-- (cofacture/soap/realsend_test.go); el resto de códigos son el catálogo oficial estándar de
-- "Tipo de Documento" de la DIAN, no confirmados con un envío propio porque ningún emisor de
-- prueba los usó todavía — el Anexo Técnico remite esta tabla a su "Caja de Herramientas"
-- (un .xlsx que no está en este repositorio, sección 13.2.7.1).
CREATE TABLE identification_types (
    code        VARCHAR(3)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE municipalities (
    code            VARCHAR(5)  PRIMARY KEY,
    name            TEXT        NOT NULL,
    department_code VARCHAR(2)  NOT NULL REFERENCES departments(code),
    description     TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_methods (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tax_types (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE unit_measures (
    code        VARCHAR(3)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tipos de documento ELECTRÓNICO DIAN — compartido por Invoice/CreditNote/DebitNote y, a
-- futuro, Documento Soporte/exportación/importación/contingencia (sección 13.2.4 del Anexo
-- Técnico 1.9: "Tipo de Documento: cbc:InvoiceTypeCode y cbc:CreditnoteTypeCode" es un solo
-- catálogo, no uno por tipo de documento). NO se llama invoice_type_codes a propósito: ese
-- nombre sugeriría que es solo de la factura, igual que cufe.go no debía contener CUDE.
--
-- A diferencia de las demás, esta no viene de un CSV del proyecto legacy: son los tres
-- códigos que ya implementa y valida cofacture (domain.DocumentTypeCode), confirmados contra
-- la DIAN real en la Fase 1. 02/03/04/05 quedan fuera hasta que cofacture los soporte.
CREATE TABLE dian_document_types (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
