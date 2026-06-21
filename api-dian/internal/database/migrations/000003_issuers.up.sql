-- Fase 2.3: issuers — configuración mínima de un emisor/tenant para emitir documentos DIAN.
-- No es un perfil de empresa completo (CRM) — ver docs/api-dian-architecture.md sección 4.2.
--
-- software_pin/certificate/certificate_password se guardan cifrados con AES-256-GCM
-- (internal/issuers/secrets.go), nunca en texto plano. La clave de cifrado viene de la
-- variable de entorno ISSUER_SECRETS_KEY, no de esta base de datos.
CREATE TABLE issuers (
    id                       UUID         PRIMARY KEY,
    nit                      VARCHAR(20)  NOT NULL UNIQUE,
    check_digit              VARCHAR(1)   NOT NULL,
    business_name            TEXT         NOT NULL,
    trade_name               TEXT,
    identification_type_code VARCHAR(3)   NOT NULL REFERENCES identification_types(code),
    department_code          VARCHAR(2)   NOT NULL REFERENCES departments(code),
    municipality_code        VARCHAR(5)   NOT NULL REFERENCES municipalities(code),
    address_line             TEXT         NOT NULL,
    email                    TEXT         NOT NULL,
    phone                    TEXT,
    environment              VARCHAR(1)   NOT NULL,
    software_id              TEXT         NOT NULL,
    software_pin             BYTEA        NOT NULL,
    certificate              BYTEA        NOT NULL,
    certificate_password     BYTEA        NOT NULL,
    is_active                BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_issuers_environment CHECK (environment IN ('1', '2'))
);

CREATE INDEX idx_issuers_nit ON issuers(nit);
