-- Ventas: esquema unificado (ventas, cotizaciones, pagos).
--
-- Consolida lo que antes eran 000001_sales/000002_quotes/000003_payments/000004_invoice_link/
-- 000005_line_discount en una sola migración — sin datos reales todavía.

CREATE SCHEMA IF NOT EXISTS sales;

CREATE TABLE IF NOT EXISTS sales.sales (
    id                   UUID PRIMARY KEY,
    company_id           UUID NOT NULL,
    customer_id          UUID NOT NULL,
    number               TEXT NOT NULL DEFAULT '',
    status               VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date           TIMESTAMPTZ NOT NULL,
    due_date             TIMESTAMPTZ,
    notes                TEXT NOT NULL DEFAULT '',
    -- Forma/medio de pago (catálogos DIAN payment_terms/payment_methods, mismo tipo que usa
    -- electronic.documents.payment_means) -- se hereda a la factura electrónica generada desde
    -- esta venta en vez de forzar "Contado/Efectivo" (ver electronic/application/from_sale.go).
    payment_means        JSONB NOT NULL DEFAULT '[]',
    -- Factura electrónica (si alguna) generada a partir de esta venta, para no poder generar dos
    -- veces desde la misma venta (ver electronic/application CreateFromSaleUseCase). Sin FK a
    -- electronic.documents a propósito: cada módulo es dueño de su schema, la integridad se
    -- maneja en la aplicación (mismo criterio que product_id en sale_lines, sin referencia
    -- cruzada a product.products).
    invoice_document_id UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sales_company ON sales.sales(company_id);
-- Único condicional: number='' identifica borradores (aún no numerados), pueden repetirse;
-- una vez confirmada la venta y asignado un consecutivo real, no puede repetirse por empresa.
CREATE UNIQUE INDEX IF NOT EXISTS uq_sales_number ON sales.sales(company_id, number) WHERE number != '';

CREATE TABLE IF NOT EXISTS sales.sale_lines (
    id          UUID PRIMARY KEY,
    sale_id     UUID NOT NULL REFERENCES sales.sales(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity    NUMERIC(18,4) NOT NULL,
    unit_price  NUMERIC(18,4) NOT NULL,
    tax_rate    NUMERIC(6,2) NOT NULL DEFAULT 0,
    -- Descuento por línea (porcentaje 0-100), aplicado antes de impuestos.
    discount    NUMERIC NOT NULL DEFAULT 0,
    subtotal    NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount  NUMERIC(18,4) NOT NULL DEFAULT 0,
    total       NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sale_lines_sale ON sales.sale_lines(sale_id);

CREATE TABLE IF NOT EXISTS sales.quotes (
    id          UUID PRIMARY KEY,
    company_id  UUID NOT NULL,
    customer_id UUID NOT NULL,
    number      TEXT NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',
    issue_date  TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    notes       TEXT NOT NULL DEFAULT '',
    payment_means JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotes_company ON sales.quotes(company_id);

CREATE TABLE IF NOT EXISTS sales.quote_lines (
    id          UUID PRIMARY KEY,
    quote_id    UUID NOT NULL REFERENCES sales.quotes(id) ON DELETE CASCADE,
    product_id  UUID NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    quantity    NUMERIC(18,4) NOT NULL,
    unit_price  NUMERIC(18,4) NOT NULL,
    tax_rate    NUMERIC(6,2) NOT NULL DEFAULT 0,
    discount    NUMERIC NOT NULL DEFAULT 0,
    subtotal    NUMERIC(18,4) NOT NULL DEFAULT 0,
    tax_amount  NUMERIC(18,4) NOT NULL DEFAULT 0,
    total       NUMERIC(18,4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_quote_lines_quote ON sales.quote_lines(quote_id);

CREATE TABLE IF NOT EXISTS sales.sale_payments (
    id             UUID PRIMARY KEY,
    company_id     UUID NOT NULL,
    sale_id        UUID NOT NULL REFERENCES sales.sales(id),
    payment_date   TIMESTAMPTZ NOT NULL,
    amount         NUMERIC(18,4) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'transfer',
    reference      TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sale_payments_sale    ON sales.sale_payments(sale_id);
CREATE INDEX IF NOT EXISTS idx_sale_payments_company ON sales.sale_payments(company_id);

-- Consecutivo interno de ventas y cotizaciones — se reinicia cada año, uno por empresa y por
-- tipo de documento (doc_type: 'sale' | 'quote'). Mismo patrón que accounting.voucher_counters.
CREATE TABLE IF NOT EXISTS sales.number_counters (
    company_id UUID        NOT NULL,
    doc_type   VARCHAR(10) NOT NULL,
    year       INTEGER     NOT NULL,
    last_seq   INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, doc_type, year)
);
