-- Módulo de moneda extranjera: tasas de cambio (TRM) y trazabilidad en asientos.

-- Tabla de tasas de cambio históricas.
-- rate_x10000 = TRM × 10 000 para evitar float64.
-- Ejemplo: 1 USD = 4 200.1234 COP → rate_x10000 = 42 001 234.
CREATE TABLE accounting.exchange_rates (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    rate_date       DATE        NOT NULL,
    from_currency   CHAR(3)     NOT NULL,
    to_currency     CHAR(3)     NOT NULL DEFAULT 'COP',
    rate_x10000     BIGINT      NOT NULL CHECK (rate_x10000 > 0),
    source          VARCHAR(20) NOT NULL DEFAULT 'MANUAL',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT exchange_rates_pkey   PRIMARY KEY (id),
    CONSTRAINT exchange_rates_unique UNIQUE (rate_date, from_currency, to_currency)
);

-- Trazabilidad de moneda extranjera en líneas de asiento.
-- foreign_amount: monto en la moneda original (centavos), siempre positivo.
-- foreign_currency: código ISO 4217 ("USD", "EUR"); NULL para asientos en COP.
ALTER TABLE accounting.journal_lines
    ADD COLUMN foreign_amount   BIGINT,
    ADD COLUMN foreign_currency CHAR(3);

-- Índice parcial para consultas de revaluación.
CREATE INDEX journal_lines_forex_idx
    ON accounting.journal_lines (foreign_currency)
    WHERE foreign_currency IS NOT NULL;
