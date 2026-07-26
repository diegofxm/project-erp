-- Activos fijos (Propiedad, Planta y Equipo — PPE).
-- Cada activo almacena las tres cuentas PUC que necesita para generar asientos:
--   asset_account        → 1504/1520/1524/1528/1536... (la cuenta del activo)
--   depreciation_account → 516005/516010/516020...     (gasto de depreciación)
--   accumulated_account  → 159205/159220/159228...     (depreciación acumulada)
-- Las cuentas de ganancia/pérdida en baja son configurables por activo; los
-- defaults corresponden a los códigos colombianos más comunes del PUC.
CREATE TABLE IF NOT EXISTS accounting.fixed_assets (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID          NOT NULL,
    code                  VARCHAR(20)   NOT NULL,
    name                  VARCHAR(200)  NOT NULL,
    description           TEXT,
    asset_account         VARCHAR(10)   NOT NULL,
    depreciation_account  VARCHAR(10)   NOT NULL,
    accumulated_account   VARCHAR(10)   NOT NULL,
    gain_account          VARCHAR(10)   NOT NULL DEFAULT '424505',
    loss_account          VARCHAR(10)   NOT NULL DEFAULT '529005',
    acquisition_date      DATE          NOT NULL,
    acquisition_cost      BIGINT        NOT NULL CHECK (acquisition_cost > 0),
    salvage_value         BIGINT        NOT NULL DEFAULT 0 CHECK (salvage_value >= 0),
    useful_life_months    INTEGER       NOT NULL CHECK (useful_life_months > 0),
    depreciation_method   VARCHAR(20)   NOT NULL DEFAULT 'STRAIGHT_LINE'
                              CHECK (depreciation_method IN ('STRAIGHT_LINE')),
    status                VARCHAR(20)   NOT NULL DEFAULT 'ACTIVE'
                              CHECK (status IN ('ACTIVE','DISPOSED','FULLY_DEPRECIATED')),
    third_party_nit       VARCHAR(20),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, code)
);

-- Registro de cada corrida de depreciación mensual.
-- La restricción UNIQUE parcial evita depreciar el mismo periodo dos veces.
CREATE TABLE IF NOT EXISTS accounting.depreciation_runs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID        NOT NULL,
    period_id   UUID        NOT NULL,
    run_date    DATE        NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'COMPLETED'
                    CHECK (status IN ('COMPLETED','REVERSED')),
    journal_id  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS depreciation_runs_period_unique_idx
    ON accounting.depreciation_runs (company_id, period_id)
    WHERE status = 'COMPLETED';

-- Detalle por activo de cada corrida: qué monto se depreció para cada uno.
-- Permite calcular la depreciación acumulada de un activo sumando sus entradas.
CREATE TABLE IF NOT EXISTS accounting.depreciation_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id     UUID        NOT NULL REFERENCES accounting.depreciation_runs(id),
    asset_id   UUID        NOT NULL REFERENCES accounting.fixed_assets(id),
    amount     BIGINT      NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS depreciation_entries_asset_idx
    ON accounting.depreciation_entries (asset_id);
