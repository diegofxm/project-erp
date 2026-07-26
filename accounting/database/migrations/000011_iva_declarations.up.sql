-- Declaraciones de IVA (Formulario 300).
-- Cada fila representa una declaración bimestral, cuatrimestral o anual.
-- El estado PAID implica que se generó el asiento de pago (journal_id).
CREATE TABLE IF NOT EXISTS accounting.iva_declarations (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL,
    period_start     DATE        NOT NULL,
    period_end       DATE        NOT NULL,
    period_type      VARCHAR(20) NOT NULL
                         CHECK (period_type IN ('BIMESTRAL','CUATRIMESTRAL','ANUAL')),
    generated_iva    BIGINT      NOT NULL DEFAULT 0,  -- IVA cobrado en ventas
    deductible_iva   BIGINT      NOT NULL DEFAULT 0,  -- IVA pagado en compras
    withheld_iva     BIGINT      NOT NULL DEFAULT 0,  -- Reteiva que nos practicaron
    net_iva          BIGINT      NOT NULL DEFAULT 0,  -- generado - descontable - retenido
    previous_balance BIGINT      NOT NULL DEFAULT 0,  -- saldo a favor período anterior
    amount_to_pay    BIGINT      NOT NULL DEFAULT 0,  -- max(0, net_iva - previous_balance)
    carry_forward    BIGINT      NOT NULL DEFAULT 0,  -- saldo a favor para el próximo período
    status           VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
                         CHECK (status IN ('DRAFT','FILED','PAID','CORRECTED')),
    journal_id       UUID,       -- asiento de pago (cuando status = 'PAID')
    filed_at         TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, period_start, period_end)
);
