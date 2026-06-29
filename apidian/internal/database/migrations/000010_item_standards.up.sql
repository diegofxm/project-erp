-- item_standards: catálogo real de la tabla 13.3.5 del Anexo Técnico 1.9 ("Productos
-- @schemeID, @schemeName, @schemeAgencyID" — ver docs/reference/Caja de herramientas
-- FE_V19_(v2026)/Anexo Tecnico/Tablas Referenciadas). Exactamente 4 filas fijas: el código del
-- ÍTEM real (UNSPSC/GTIN/Partida Arancelaria/propio) NO se valida contra un catálogo completo
-- aquí (esos son externos, de decenas de miles de códigos, mismo tratamiento que CIIU) — lo
-- que sí es un catálogo chico y cerrado es el SELECTOR de qué estándar se está usando, y la
-- DIAN rechaza explícitamente si @schemeID/@schemeName/@schemeAgencyID no coinciden con esta
-- tabla (ver docs/apidian-architecture.md sección 9.45).
--
-- agency_id es '' (no NULL) para la fila "999" (estándar propio del contribuyente), que exige
-- explícitamente que @schemeAgencyID NO se use en el XML — mismo criterio que el resto del
-- proyecto, que modela "ausente" como string vacío en vez de *string/NULL (ver
-- domain.Line.ItemTypeAgencyID, un string plano) — cofacture/builder omite el atributo por
-- completo cuando el valor viene vacío (ver line_items.go).
CREATE TABLE item_standards (
    code        VARCHAR(10) PRIMARY KEY,
    name        TEXT        NOT NULL,
    agency_id   VARCHAR(10) NOT NULL DEFAULT '',
    description TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
