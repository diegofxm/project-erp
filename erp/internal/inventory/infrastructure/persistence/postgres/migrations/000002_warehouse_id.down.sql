ALTER TABLE inventory.movements DROP COLUMN transfer_group_id;
ALTER TABLE inventory.movements DROP COLUMN warehouse_id;
ALTER TABLE inventory.movements ADD COLUMN warehouse TEXT NOT NULL DEFAULT 'principal';

ALTER TABLE inventory.stock DROP CONSTRAINT IF EXISTS uq_stock_company_product_warehouse;
ALTER TABLE inventory.stock DROP COLUMN warehouse_id;
ALTER TABLE inventory.stock ADD COLUMN warehouse TEXT NOT NULL DEFAULT 'principal';
ALTER TABLE inventory.stock ADD CONSTRAINT stock_company_id_product_id_warehouse_key UNIQUE (company_id, product_id, warehouse);
