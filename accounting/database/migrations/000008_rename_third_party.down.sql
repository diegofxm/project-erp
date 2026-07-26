ALTER TABLE accounting.journal_lines
    RENAME COLUMN third_party_nit TO tercero_nit;

ALTER INDEX IF EXISTS accounting.journal_lines_third_party_nit_idx
    RENAME TO journal_lines_tercero_idx;
