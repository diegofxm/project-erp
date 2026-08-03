-- Reemplaza la bodega de texto libre ("principal" hardcodeado en el código Go) por una
-- referencia real a company.warehouses.id — sin FK a nivel de base de datos porque cada módulo
-- es dueño de su propio schema (mismo criterio que en el resto del proyecto), la integridad se
-- garantiza en la aplicación (ver inventory application, que resuelve la bodega vía
-- company.WarehouseRepository.GetOrCreateDefault antes de escribir cualquier movimiento).
--
-- Se limpian las filas existentes en vez de migrar el texto libre: son datos de prueba de
-- desarrollo, no hay clientes reales todavía (ver feedback_migrations_vs_seed / feedback_sql_
-- schema_conventions — unificar mientras no haya datos reales).
DELETE FROM inventory.movements;
DELETE FROM inventory.stock;

ALTER TABLE inventory.stock DROP COLUMN warehouse CASCADE;
ALTER TABLE inventory.stock ADD COLUMN warehouse_id UUID NOT NULL;
ALTER TABLE inventory.stock ADD CONSTRAINT uq_stock_company_product_warehouse UNIQUE (company_id, product_id, warehouse_id);

ALTER TABLE inventory.movements DROP COLUMN warehouse CASCADE;
ALTER TABLE inventory.movements ADD COLUMN warehouse_id UUID NOT NULL;
ALTER TABLE inventory.movements ADD COLUMN transfer_group_id UUID;

CREATE INDEX IF NOT EXISTS idx_movements_transfer_group ON inventory.movements(transfer_group_id) WHERE transfer_group_id IS NOT NULL;
