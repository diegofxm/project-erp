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

-- documents.customer_id: referencia OPCIONAL y de solo trazabilidad a este catálogo — NUNCA
-- la fuente de verdad del documento (eso sigue siendo documents.customer, el snapshot JSONB
-- que de verdad se firmó y se envió a la DIAN), ni se serializa en ningún XML — no afecta el
-- cumplimiento del Anexo Técnico. Va aquí (no en 000005_documents, donde se crea la tabla
-- documents) porque en ese punto de la secuencia customers todavía no existe — la FK fallaría
-- en una instalación nueva desde cero si se intentara antes de este punto.
-- Nullable (una factura con datos sueltos, sin cliente guardado, no la tiene) y
-- ON DELETE SET NULL (borrar un cliente nunca debe romper ni borrar documentos ya emitidos).
ALTER TABLE documents ADD COLUMN customer_id UUID REFERENCES customers(id) ON DELETE SET NULL;

CREATE INDEX idx_documents_customer ON documents(customer_id) WHERE customer_id IS NOT NULL;
