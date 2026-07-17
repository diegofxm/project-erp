-- Fase NE: credenciales separadas de software para Nómina Electrónica.
-- FE y DS comparten el mismo Software ID/PIN (confirmado en el portal DIAN — al habilitar DS
-- el sistema reutiliza el mismo software de FE y solo entrega un TestSetID distinto, que ya
-- vive en numbering_ranges.test_set_id). NE sí tiene software propio en el portal DIAN.
--
-- Nullable: un emisor puede configurar NE en cualquier momento (mismo patrón gradual que
-- software_id/software_pin — ver docs/apidian-architecture.md sección 9.25).
-- ne_software_pin es BYTEA — se cifra con AES-256-GCM antes de guardar.
ALTER TABLE issuers
    ADD COLUMN ne_software_id  TEXT,
    ADD COLUMN ne_software_pin BYTEA;
