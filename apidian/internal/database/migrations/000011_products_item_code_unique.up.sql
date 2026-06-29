-- item_code no tenía ninguna restricción de unicidad: dos productos del mismo emisor podían
-- guardarse con el mismo código sin ningún aviso (ver products.Service.validateProduct, que
-- nunca lo comprobaba). Índice parcial porque item_code es opcional — nullableString() ya
-- convierte "" a NULL al guardar (ver internal/products/postgres.go), así que varios productos
-- sin código conviven sin chocar entre sí; solo se exige unicidad entre los que SÍ tienen uno,
-- y solo dentro del mismo emisor (un mismo código de ítem puede repetirse entre emisores
-- distintos sin problema).
CREATE UNIQUE INDEX idx_products_issuer_item_code ON products(issuer_id, item_code) WHERE item_code IS NOT NULL;
