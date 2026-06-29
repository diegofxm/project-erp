-- Fase 2.11: products — catálogo de ítems/servicios reutilizable, por emisor. Misma lógica de
-- conveniencia que customers (ver 000005_customers): no es la fuente de verdad de una línea ya
-- emitida, esa sigue viviendo en documents.lines (JSONB).
--
-- No incluye quantity/line_extension_cents ni una lista de impuestos — eso es dato de USO
-- (cuántas unidades, en qué factura), no de catálogo. tax_type_code/tax_type_name/tax_percent
-- son un único impuesto por defecto, de conveniencia; si una factura real necesita más de un
-- impuesto por línea, eso se decide al construir esa línea, no aquí.
--
-- unit_code NO es FK a unit_measures (aunque desde la sección 9.46 ese catálogo ya está
-- completo — 1.081 códigos reales UN/ECE Rec. 20 — y técnicamente ya podría serlo):
-- domain.Line.UnitCode (el snapshot real que viaja en documents.lines) vive en JSONB, donde un
-- FK nunca es posible — agregar el FK aquí pero no ahí sería una validación a medias, más
-- confusa que útil. Si algún día se valida unit_code en serio, debe ser en código (igual
-- patrón que IsValidPaymentTerm/IsValidLiabilityCode), no con un FK que solo cubre la mitad
-- del dato.
CREATE TABLE products (
    id                   UUID         PRIMARY KEY,
    issuer_id           UUID         NOT NULL REFERENCES issuers(id),
    description        TEXT         NOT NULL,
    unit_code         VARCHAR(3)   NOT NULL,
    unit_price_cents BIGINT       NOT NULL DEFAULT 0,
    item_code       TEXT,
    item_type_code TEXT,
    item_type_name TEXT,
    item_type_agency_id TEXT,
    tax_type_code        VARCHAR(2) REFERENCES tax_types(code),
    tax_type_name         TEXT,
    tax_percent            DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_issuer ON products(issuer_id);
