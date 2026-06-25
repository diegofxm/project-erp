-- Fase 2.2: esquema de catálogos DIAN — solo estructura. Los datos se cargan por código
-- desde internal/database/seed/*.csv (ver database.Seed), no aquí: catálogos como
-- municipalities tienen cientos de filas reales y no escalan como INSERT literal en una
-- migración, y además así se pueden actualizar sin escribir una migración nueva cada vez.
--
-- Alcance recortado al de este orquestador (Invoice 01 / CreditNote 91 / DebitNote 92, ver
-- docs/apidian-architecture.md sección 9.2): no incluye catálogos de Documento Soporte,
-- Tiquete POS, Documentos Equivalentes ni Nómina — esos no aplican todavía.
--
-- 2026-06-24: se consiguió la "Caja de Herramientas Factura Electrónica" oficial de la DIAN
-- (docs/reference/Caja de herramientas FE_V19_(v2026)/) — de ahí salen `payment_terms`/
-- `tax_regimes`/`liability_codes` (antes no existían ni como tabla) y se completaron con
-- datos reales `municipalities`/`departments`/`tax_types`/`payment_methods` (ver
-- docs/apidian-architecture.md sección 9.34). `type_organizations` (Persona Natural/Jurídica)
-- NO tiene tabla a propósito: son solo 2 valores fijos ("1"/"2"), confirmados completos
-- contra el catálogo oficial, y ya se validan en código (issuers.defaultEntityTypeCode) — una
-- tabla de 2 filas no agrega nada. CIIU (actividad económica) tampoco es tabla: es catálogo
-- de la DANE, no de la DIAN, se modela como array libre sin catálogo (ver issuers.Issuer).

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
-- prueba los usó todavía. Contrastado contra el `.sch` de validación real de la Caja de
-- Herramientas (sección 13.2.7.1, fechado 2019): esa lista no incluye "47" (NIT de otro
-- país) que sí tenemos — se deja como está a propósito (no se quita sin evidencia más
-- reciente de que ya no es válido; el `.sch` tiene al menos una contradicción confirmada más
-- abajo en liability_codes — así que no se trata como verdad absoluta).
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

-- Medios de Pago (cbc:PaymentMeansCode) — no confundir con payment_terms (Formas de Pago,
-- cbc:PaymentMeans/ID, más abajo), son catálogos distintos del Anexo Técnico. code llega a
-- VARCHAR(3) porque el código catch-all oficial es "ZZZ", no "ZZ".
CREATE TABLE payment_methods (
    code        VARCHAR(3)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Formas de Pago (cbc:PaymentMeans/ID) — NO confundir con payment_methods (Medios de Pago,
-- cbc:PaymentMeansCode, arriba): son catálogos distintos del Anexo Técnico. Solo 2 valores
-- oficiales ("1" Contado, "2" Crédito) — domain.PaymentMean.Code en cofacture ya distinguía
-- este campo del medio de pago, pero hasta ahora no tenía catálogo detrás.
CREATE TABLE payment_terms (
    code        VARCHAR(1)  PRIMARY KEY,
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

-- Tipo de Régimen — va como atributo listName de cbc:TaxLevelCode (domain.Party.TaxRegimeCode
-- en cofacture, que ya existía pero nunca se poblaba desde apidian). La Caja de Herramientas
-- no tiene un archivo .gc dedicado para este catálogo (a diferencia de los demás) — los
-- valores vienen únicamente del Schematron de validación real (DIAN_UBL21-listacodigos_v1.6.
-- sch, regla TipoRegimen), que no trae nombres legibles, solo los códigos válidos. Por eso
-- "name"/"description" no son textos oficiales — se deja explícito en el seed (ver
-- seed/tax_regimes.csv) en vez de inventar nombres con apariencia de oficiales.
CREATE TABLE tax_regimes (
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

-- Responsabilidades fiscales (cbc:TaxLevelCode) — hasta ahora solo existía el valor por
-- defecto "R-99-PN" en código (issuers.Service/documents.Service), sin catálogo real detrás.
-- El Schematron real de la DIAN valida contra una lista mucho más larga (códigos O-xx/A-xx/
-- E-xx/R-xx) que la Caja de Herramientas no nombra por completo — aquí se cargan los 5
-- códigos que SÍ tienen nombre oficial confirmado en responsabilidad-2.1.gc (Gran
-- contribuyente, Autorretenedor, Agente de retención IVA, Régimen simple, catch-all
-- "No aplica" = R-99-PJ) MÁS "R-99-PN", el código catch-all que de verdad se usa hoy como
-- default. No se inventan nombres para el resto de códigos válidos del Schematron.
--
-- Discrepancia ya RESUELTA con evidencia real (2026-06-24): el Schematron (2019) solo lista
-- "R-99-PJ" como catch-all, no "R-99-PN" — pero una factura real (SETP990300000, NIT
-- 6382356, persona natural) usando "R-99-PN" fue autorizada por la DIAN real (StatusCode 00,
-- "Procesado Correctamente"). Confirma que el Schematron de 2019 está desactualizado en este
-- punto y que el default actual del código ("R-99-PN") es correcto — se mantiene sin
-- cambios. Se dejan ambos códigos en el catálogo (R-99-PJ también es válido, por si algún
-- emisor persona jurídica lo necesita).
CREATE TABLE liability_codes (
    code        VARCHAR(10) PRIMARY KEY,
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
-- la DIAN real en la Fase 1. El `.sch` de la Caja de Herramientas confirma que el catálogo
-- vigente real es 01/02/03/91/92 — 02/03 quedan fuera a propósito hasta que cofacture los
-- soporte: un catálogo no debe aceptar un tipo de documento que el builder no sabe construir.
CREATE TABLE dian_document_types (
    code        VARCHAR(2)  PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
