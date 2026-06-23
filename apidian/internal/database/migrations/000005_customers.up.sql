-- Fase 2.11: customers — catálogo de adquirientes reutilizable, por emisor. Conveniencia para
-- el frontend (no retipear los mismos datos en cada factura), NUNCA la fuente de verdad de un
-- documento ya emitido: cada documento sigue guardando su propio snapshot en
-- documents.customer (JSONB) tal cual llegó al emitir — borrar o editar un customer aquí no
-- afecta documentos ya emitidos.
--
-- Mismas columnas aplanadas que domain.Party (identification/address/tax scheme/etc.) — igual
-- criterio que issuers, no una tabla anidada para la dirección. A diferencia de issuers,
-- identification_type_code SÍ es FK (catálogo genérico de terceros), pero address_city_code/
-- address_state_code NO lo son: un cliente puede ser extranjero (factura de exportación), sin
-- código DIVIPOLA válido.
--
-- Esta migración va ANTES que 000008_documents a propósito (renumerada el 2026-06-23, sin
-- datos reales en juego): documents.customer_id referencia customers(id), y así puede crearse
-- inline en su CREATE TABLE, en su posición lógica, en vez de un ALTER TABLE posterior — un
-- ALTER siempre agrega la columna al final físicamente, violando la convención de
-- created_at/updated_at siempre al final (ver docs/apidian-architecture.md sección 9.31).
CREATE TABLE customers (
    id                                UUID         PRIMARY KEY,
    issuer_id                        UUID         NOT NULL REFERENCES issuers(id),
    entity_type_code                 VARCHAR(1),
    identification_number            TEXT         NOT NULL,
    identification_type_code         VARCHAR(3)   NOT NULL REFERENCES identification_types(code),
    identification_verification_code VARCHAR(1),
    name                              TEXT        NOT NULL,
    address_line                      TEXT,
    address_city_code                 TEXT,
    address_city_name                  TEXT,
    address_state_code                  TEXT,
    address_state_name                   TEXT,
    address_country_code                  TEXT,
    address_country_name                   TEXT,
    tax_scheme_code                         VARCHAR(2) REFERENCES tax_types(code),
    tax_scheme_name                          TEXT,
    liability_codes                           TEXT[]   NOT NULL DEFAULT '{}',
    phone                                       TEXT,
    email                                        TEXT,
    merchant_registration_number                  TEXT,
    created_at                                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customers_issuer ON customers(issuer_id);
