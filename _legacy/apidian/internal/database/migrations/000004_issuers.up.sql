-- Emisores/tenants: empresas que emiten documentos electrónicos DIAN.
-- Todas las FKs de catálogo apuntan a catalogs.* (no más referencias a public.tax_types etc.).
--
-- liability_codes TEXT[]: Postgres no soporta FK contra cada elemento de un array;
--   se valida en issuers.Service contra catalogs.liability_codes.
-- industry_classification_codes TEXT[]: catálogo CIIU (DANE), mismo tratamiento.
--
-- software_pin / certificate / certificate_password: cifrados AES-256-GCM antes de guardar.
-- ne_software_pin: ídem, para Nómina Electrónica (software independiente del FE/DS).
-- logo / logo_content_type: representación gráfica en PDF; nullable, no cifrado.
--
-- environment: '1' = producción DIAN, '2' = habilitación (pruebas).

CREATE TABLE issuers (
    id                            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nit                           VARCHAR(20) NOT NULL UNIQUE,
    check_digit                   VARCHAR(1)  NOT NULL,
    business_name                 TEXT        NOT NULL,
    trade_name                    TEXT,
    identification_type_code      VARCHAR(3)  NOT NULL REFERENCES catalogs.identification_types(code),
    department_code               VARCHAR(2)  NOT NULL REFERENCES catalogs.departments(code),
    municipality_code             VARCHAR(5)  NOT NULL REFERENCES catalogs.municipalities(code),
    address_line                  TEXT        NOT NULL,
    email                         TEXT        NOT NULL,
    phone                         TEXT,
    environment                   VARCHAR(1)  NOT NULL,
    entity_type_code              VARCHAR(1)  NOT NULL DEFAULT '1',
    tax_scheme_code               VARCHAR(2)  NOT NULL DEFAULT 'ZZ' REFERENCES catalogs.dian_tax_types(code),
    tax_scheme_name               TEXT        NOT NULL DEFAULT 'No aplica',
    liability_codes               TEXT[]      NOT NULL DEFAULT '{}',
    tax_regime_code               VARCHAR(2)  REFERENCES catalogs.tax_regimes(code),
    industry_classification_codes TEXT[]      NOT NULL DEFAULT '{}',
    merchant_registration_number  TEXT,
    software_id                   TEXT,
    software_pin                  BYTEA,
    certificate                   BYTEA,
    certificate_password          BYTEA,
    ne_software_id                TEXT,
    ne_software_pin               BYTEA,
    logo                          BYTEA,
    logo_content_type             VARCHAR(10),
    email_body_template           TEXT,
    is_active                     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_issuers_environment CHECK (environment IN ('1', '2'))
);

CREATE INDEX idx_issuers_nit ON issuers(nit);
