-- Marcas de conciliación de cartera.
-- Cada fila empareja una línea de débito (factura/cargo) con su línea de crédito
-- (pago/abono). La restricción UNIQUE impide reconciliar una línea dos veces.
-- Las líneas del ledger siguen siendo inmutables; la conciliación es un overlay.
CREATE TABLE IF NOT EXISTS accounting.reconciliation_marks (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL,
    journal_line_id  UUID        NOT NULL REFERENCES accounting.journal_lines(id),
    reconciled_with  UUID        REFERENCES accounting.journal_lines(id),
    note             TEXT,
    reconciled_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (journal_line_id)
);

CREATE INDEX IF NOT EXISTS reconciliation_marks_company_idx
    ON accounting.reconciliation_marks (company_id);
