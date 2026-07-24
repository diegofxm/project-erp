-- NC (91), ND (92) y NA (95) no requieren resolución de numeración DIAN — solo FE (01) y
-- DS (05) la necesitan. Quitar NOT NULL de los cuatro campos de resolución para que los
-- documentos de ajuste puedan registrar rangos sin datos de resolución.
ALTER TABLE numbering_ranges
    ALTER COLUMN resolution_number DROP NOT NULL,
    ALTER COLUMN resolution_date   DROP NOT NULL,
    ALTER COLUMN valid_from        DROP NOT NULL,
    ALTER COLUMN valid_to          DROP NOT NULL;
