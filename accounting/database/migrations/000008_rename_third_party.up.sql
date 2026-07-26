-- Renombra tercero_nit → third_party_nit para mantener los nombres de columnas en inglés.
-- PostgreSQL actualiza automáticamente las referencias internas del índice al renombrar la columna;
-- solo hay que renombrar el índice para que el nombre refleje el nuevo nombre de columna.
ALTER TABLE accounting.journal_lines
    RENAME COLUMN tercero_nit TO third_party_nit;

ALTER INDEX IF EXISTS accounting.journal_lines_tercero_idx
    RENAME TO journal_lines_third_party_nit_idx;
