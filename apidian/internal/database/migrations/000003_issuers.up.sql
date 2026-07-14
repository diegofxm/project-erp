-- Fase 2.3/2.5: issuers — configuración mínima de un emisor/tenant para emitir documentos
-- DIAN. No es un perfil de empresa completo (CRM) — ver docs/apidian-architecture.md
-- sección 4.2.
--
-- Los campos de entity_type_code en adelante (hasta merchant_registration_number) son los
-- necesarios para construir cac:AccountingSupplierParty/cac:Party del XML
-- (cofacture/domain.Party) — se descubrieron faltantes al construir el primer Invoice real en
-- internal/documents, reproduciendo la factura que la DIAN ya autorizó en la Fase 1.7.
-- Originalmente se agregaron en una migración aparte (000005_issuers_party_fields) y se
-- unificaron aquí el 2026-06-21, sin datos reales en juego todavía.
--
-- liability_codes (responsabilidades fiscales, ej. "R-99-PN") sigue siendo TEXT[] sin FK a
-- propósito, aunque desde la migración 000002_catalogs ya existe el catálogo liability_codes:
-- Postgres no soporta un FK contra cada elemento de un array nativamente — cada código se
-- valida en issuers.Service contra ese catálogo, no a nivel de esquema.
--
-- tax_regime_code (FK a tax_regimes, migración 000002_catalogs — por eso los catálogos van
-- ANTES que issuers en la secuencia, no después) e industry_classification_codes
-- (CIIU) se agregaron el 2026-06-24 — CIIU es catálogo de la DANE, no de la DIAN, por eso es
-- TEXT[] sin FK, igual que liability_codes (validación de cardinalidad máx. 4 en
-- issuers.Service, no en el esquema).
--
-- software_pin/certificate/certificate_password se guardan cifrados con AES-256-GCM
-- (internal/issuers/secrets.go), nunca en texto plano. La clave de cifrado viene de la
-- variable de entorno ISSUER_SECRETS_KEY, no de esta base de datos.
--
-- software_id/software_pin/certificate/certificate_password son NULLABLE porque el registro
-- inicial (POST /auth/register) ya no los exige — el "espacio de empresa" (NIT, razón
-- social, dirección) se completa primero, y software/certificado se cargan después,
-- independientemente, vía PUT /issuers/me, en el orden en que el usuario los vaya
-- consiguiendo (ver docs/apidian-architecture.md sección 9.25). documents.Service los exige
-- recién al confirmar un documento (ErrIssuerNotReadyToIssue si faltan), nunca antes.
--
-- logo/logo_content_type (agregados para la representación gráfica en PDF, ver
-- docs/apidian-architecture.md sección 9.39): NULLABLE, opcionales — un emisor sin logo
-- todavía debe poder emitir y generar su PDF sin esa imagen. No son secretos, no se cifran.
--
-- created_at/updated_at van al final de la tabla por convención en todo este esquema (ver
-- docs/apidian-architecture.md sección 4.1) — nunca intercalados entre columnas de negocio.
CREATE TABLE issuers (
    id                            UUID         PRIMARY KEY,
    nit                           VARCHAR(20)  NOT NULL UNIQUE,
    check_digit                   VARCHAR(1)   NOT NULL,
    business_name                 TEXT        NOT NULL,
    trade_name                    TEXT,
    identification_type_code      VARCHAR(3)  NOT NULL REFERENCES identification_types(code),
    department_code               VARCHAR(2)  NOT NULL REFERENCES departments(code),
    municipality_code             VARCHAR(5)  NOT NULL REFERENCES municipalities(code),
    address_line                  TEXT        NOT NULL,
    email                         TEXT        NOT NULL,
    phone                         TEXT,
    environment                   VARCHAR(1)  NOT NULL,
    entity_type_code               VARCHAR(1) NOT NULL DEFAULT '1',
    tax_scheme_code                 VARCHAR(2) NOT NULL DEFAULT 'ZZ' REFERENCES tax_types(code),
    tax_scheme_name                  TEXT      NOT NULL DEFAULT 'No aplica',
    liability_codes                   TEXT[]   NOT NULL DEFAULT '{}',
    tax_regime_code                VARCHAR(2) REFERENCES tax_regimes(code),
    industry_classification_codes     TEXT[]   NOT NULL DEFAULT '{}',
    merchant_registration_number       TEXT,
    software_id                  TEXT,
    software_pin                 BYTEA,
    certificate                  BYTEA,
    certificate_password         BYTEA,
    logo                          BYTEA,
    logo_content_type             VARCHAR(10),
    email_body_template           TEXT,
    is_active                    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_issuers_environment CHECK (environment IN ('1', '2'))
);

CREATE INDEX idx_issuers_nit ON issuers(nit);
