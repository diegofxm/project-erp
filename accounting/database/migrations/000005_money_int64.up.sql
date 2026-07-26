-- Migra los campos monetarios de NUMERIC(18,2) a BIGINT (centavos).
-- Los valores existentes se multiplican por 100 para preservar la precisión.
ALTER TABLE accounting.journal_lines
  ALTER COLUMN debit  TYPE BIGINT USING ROUND(debit  * 100)::BIGINT,
  ALTER COLUMN credit TYPE BIGINT USING ROUND(credit * 100)::BIGINT;

ALTER TABLE accounting.bank_statement_lines
  ALTER COLUMN debit  TYPE BIGINT USING ROUND(debit  * 100)::BIGINT,
  ALTER COLUMN credit TYPE BIGINT USING ROUND(credit * 100)::BIGINT;
