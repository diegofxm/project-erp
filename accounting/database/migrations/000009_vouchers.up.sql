-- Tipos de comprobante que cada empresa configura (CE, CI, NC, NI, etc.).
-- Los tipos de sistema (CJ, AP) no necesitan registro previo — el contador
-- funciona independientemente de esta tabla.
CREATE TABLE IF NOT EXISTS accounting.voucher_types (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID         NOT NULL,
    code            VARCHAR(10)  NOT NULL,
    name            VARCHAR(100) NOT NULL,
    resets_annually BOOLEAN      NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, code)
);

-- Contador atómico de consecutivos por empresa, tipo y año.
-- Cuando resets_annually = false el año se fija a 0 para mantener una sola secuencia.
CREATE TABLE IF NOT EXISTS accounting.voucher_counters (
    company_id UUID         NOT NULL,
    code       VARCHAR(10)  NOT NULL,
    year       INTEGER      NOT NULL,
    last_seq   INTEGER      NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, code, year)
);

-- Columnas en journal_entries para el comprobante asignado.
ALTER TABLE accounting.journal_entries
    ADD COLUMN IF NOT EXISTS voucher_type   VARCHAR(10),
    ADD COLUMN IF NOT EXISTS voucher_number VARCHAR(30);

CREATE INDEX IF NOT EXISTS journal_entries_voucher_number_idx
    ON accounting.journal_entries (company_id, voucher_number)
    WHERE voucher_number IS NOT NULL;
