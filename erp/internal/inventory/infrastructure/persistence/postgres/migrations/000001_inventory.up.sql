-- Inventario: esquema unificado (existencias, movimientos).
--
-- Consolida lo que antes eran 000001_inventory/000002_warehouse_id en una sola migración — sin
-- datos reales todavía. La bodega ya nace como referencia a company.warehouses.id (sin FK a
-- nivel de base de datos porque cada módulo es dueño de su propio schema — mismo criterio que
-- en el resto del proyecto —, la integridad se garantiza en la aplicación: ver inventory
-- application, que resuelve la bodega vía company.WarehouseRepository.GetOrCreateDefault antes
-- de escribir cualquier movimiento).

CREATE SCHEMA IF NOT EXISTS inventory;

-- Stock actual por producto y bodega
CREATE TABLE IF NOT EXISTS inventory.stock (
    id           UUID PRIMARY KEY,
    company_id   UUID NOT NULL,
    product_id   UUID NOT NULL,
    warehouse_id UUID NOT NULL,
    quantity     NUMERIC(18,4) NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (company_id, product_id, warehouse_id)
);

CREATE INDEX IF NOT EXISTS idx_stock_company ON inventory.stock(company_id);

-- Historial de movimientos
CREATE TABLE IF NOT EXISTS inventory.movements (
    id                UUID PRIMARY KEY,
    company_id        UUID NOT NULL,
    number            VARCHAR(30) NOT NULL DEFAULT '',
    product_id        UUID NOT NULL,
    warehouse_id      UUID NOT NULL,
    type              VARCHAR(20) NOT NULL,  -- entry, exit, transfer, adjust
    quantity          NUMERIC(18,4) NOT NULL,
    reference         TEXT NOT NULL DEFAULT '',
    description       TEXT NOT NULL DEFAULT '',
    transfer_group_id UUID,
    -- is_addition: true si este movimiento SUMÓ al stock (entry, adjust, lado destino de un
    -- transfer), false si RESTÓ (exit, lado origen de un transfer). "type" no alcanza para saber
    -- el signo en un transfer -- las dos filas del mismo transfer_group_id comparten type=transfer
    -- pero una resta y la otra suma. Se usa para revertir el efecto en stock al eliminar un
    -- movimiento (ver Repository.DeleteMovement) sin tener que adivinar la dirección.
    is_addition       BOOLEAN NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_movements_company        ON inventory.movements(company_id);
CREATE INDEX IF NOT EXISTS idx_movements_product        ON inventory.movements(company_id, product_id);
CREATE INDEX IF NOT EXISTS idx_movements_transfer_group ON inventory.movements(transfer_group_id) WHERE transfer_group_id IS NOT NULL;

-- Consecutivo de movimientos, uno por empresa/tipo/año (ENT-/SAL-/TRA-/AJU-) — mismo patrón que
-- sales.number_counters.
CREATE TABLE IF NOT EXISTS inventory.number_counters (
    company_id UUID        NOT NULL,
    doc_type   VARCHAR(10) NOT NULL,
    year       INTEGER     NOT NULL,
    last_seq   INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, doc_type, year)
);
