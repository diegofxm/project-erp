-- Compras: esquema unificado (órdenes, líneas, pagos, retenciones).
--
-- Consolida lo que antes eran 000001_purchase/000002_discount_and_support_doc/000003_payments/
-- 000004_withholdings en una sola migración — sin datos reales todavía.

CREATE SCHEMA IF NOT EXISTS purchase;

CREATE TABLE IF NOT EXISTS purchase.orders (
    id                   UUID PRIMARY KEY,
    company_id           UUID NOT NULL,
    supplier_id          UUID NOT NULL,
    number               TEXT NOT NULL DEFAULT '',
    status               VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date           TIMESTAMPTZ NOT NULL,
    due_date             TIMESTAMPTZ,
    notes                TEXT NOT NULL DEFAULT '',
    -- Forma/medio de pago (catálogos DIAN payment_terms/payment_methods, mismo tipo que usa
    -- electronic.documents.payment_means) -- se hereda al Documento Soporte generado desde esta
    -- orden en vez de forzar "Contado/Efectivo" (ver electronic/application/from_purchase.go).
    payment_means        JSONB NOT NULL DEFAULT '[]',
    -- Documento Soporte (si alguno) generado a partir de esta orden, para no poder generar dos
    -- veces desde la misma orden. Sin FK a electronic.documents a propósito: cada módulo es dueño
    -- de su schema, la integridad se maneja en la aplicación (mismo criterio que
    -- sales.sales.invoice_document_id).
    support_document_id UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_company ON purchase.orders(company_id);
-- Único condicional: number='' identifica borradores (aún no numerados), pueden repetirse;
-- una vez confirmada la orden y asignado un consecutivo real, no puede repetirse por empresa.
CREATE UNIQUE INDEX IF NOT EXISTS uq_purchase_orders_number ON purchase.orders(company_id, number) WHERE number != '';

CREATE TABLE IF NOT EXISTS purchase.order_lines (
    id                UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase.orders(id) ON DELETE CASCADE,
    product_id        UUID NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    quantity          NUMERIC(18,4) NOT NULL,
    unit_price        NUMERIC(18,4) NOT NULL,
    tax_rate          NUMERIC(6,2) NOT NULL DEFAULT 0,
    -- Descuento por línea (porcentaje 0-100, aplicado antes de impuestos) — igual que sales.sale_lines.
    discount          NUMERIC NOT NULL DEFAULT 0,
    subtotal          NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount        NUMERIC(18,4) NOT NULL DEFAULT 0,
    total             NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_purchase_lines_order ON purchase.order_lines(purchase_order_id);

CREATE TABLE IF NOT EXISTS purchase.purchase_payments (
    id             UUID PRIMARY KEY,
    company_id     UUID NOT NULL,
    purchase_id    UUID NOT NULL REFERENCES purchase.orders(id),
    payment_date   TIMESTAMPTZ NOT NULL,
    amount         NUMERIC(18,4) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'transfer',
    reference      TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_payments_purchase ON purchase.purchase_payments(purchase_id);
CREATE INDEX IF NOT EXISTS idx_purchase_payments_company  ON purchase.purchase_payments(company_id);

CREATE TABLE IF NOT EXISTS purchase.purchase_withholdings (
    id                UUID PRIMARY KEY,
    purchase_order_id UUID NOT NULL REFERENCES purchase.orders(id),
    concept_code      VARCHAR(20) NOT NULL,
    concept_name      TEXT NOT NULL,
    base              NUMERIC(18,4) NOT NULL,
    rate_bp           INT NOT NULL,
    amount            NUMERIC(18,4) NOT NULL,
    account_payable   VARCHAR(20) NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_purchase_withholdings_order ON purchase.purchase_withholdings(purchase_order_id);

-- Consecutivo interno de órdenes de compra — se reinicia cada año, uno por empresa. Mismo patrón
-- que accounting.voucher_counters / sales.number_counters.
CREATE TABLE IF NOT EXISTS purchase.number_counters (
    company_id UUID    NOT NULL,
    year       INTEGER NOT NULL,
    last_seq   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, year)
);
