-- Punto de reorden ("stock mínimo") — usado por el módulo inventory para resaltar existencias
-- bajas en la pantalla de stock. 0 = sin umbral configurado.
ALTER TABLE product.products ADD COLUMN IF NOT EXISTS min_stock NUMERIC NOT NULL DEFAULT 0;
