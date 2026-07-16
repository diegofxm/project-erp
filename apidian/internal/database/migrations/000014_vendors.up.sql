-- Catálogo de proveedores/terceros no obligados a facturar, por emisor. Misma estructura que
-- customers (ver 000005_customers.up.sql): columnas aplanadas de domain.Party, snapshot
-- independiente del vendor que viaja en el Documento Soporte (documents.vendor JSONB). Editar
-- o borrar un vendor no afecta documentos ya emitidos.
--
-- Va ANTES que 000015_documents_vendor_id porque documents.vendor_id referencia vendors(id).
CREATE TABLE vendors (
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
    tax_regime_code                            VARCHAR(2) REFERENCES tax_regimes(code),
    phone                                       TEXT,
    email                                        TEXT,
    merchant_registration_number                  TEXT,
    created_at                                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vendors_issuer ON vendors(issuer_id);
