-- Agrega el NIT del tercero a cada línea de asiento.
-- Obligatorio en Colombia para Medios Magnéticos / Información Exógena (DIAN).
-- Nullable: los asientos de apertura/cierre y ajustes internos no tienen tercero.
ALTER TABLE accounting.journal_lines
  ADD COLUMN IF NOT EXISTS tercero_nit VARCHAR(20);

CREATE INDEX IF NOT EXISTS journal_lines_tercero_idx ON accounting.journal_lines (tercero_nit)
  WHERE tercero_nit IS NOT NULL;
