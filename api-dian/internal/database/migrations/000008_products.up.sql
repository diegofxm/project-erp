-- Fase 2.11: products — catálogo de ítems/servicios reutilizable, por emisor. Misma lógica de
-- conveniencia que customers (ver 000007_customers): no es la fuente de verdad de una línea ya
-- emitida, esa sigue viviendo en documents.lines (JSONB).
--
-- No incluye quantity/line_extension_cents ni una lista de impuestos — eso es dato de USO
-- (cuántas unidades, en qué factura), no de catálogo. tax_type_code/tax_type_name/tax_percent
-- son un único impuesto por defecto, de conveniencia; si una factura real necesita más de un
-- impuesto por línea, eso se decide al construir esa línea, no aquí.
--
-- unit_code NO es FK a unit_measures: ese catálogo solo tiene 10 códigos de muestra (NIU/KGM/
-- LTR...) frente al estándar real completo (UN/ECE Rec. 20, cientos de códigos) — mismo hueco
-- de datos ya conocido que departments/municipalities (ver sección 9.6 del architecture doc).
-- Confirmado real: "94" (el código que usan los tests contra la DIAN real, autorizado con
-- StatusCode 00) no está en esas 10 filas — un FK aquí habría bloqueado un producto válido.
-- domain.Line.UnitCode tampoco se valida contra catálogo (vive en JSONB sin FK); esta columna
-- sigue el mismo criterio en vez de inventar un catálogo completo sin la fuente oficial.
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
