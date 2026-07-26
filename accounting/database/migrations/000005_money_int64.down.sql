ALTER TABLE accounting.journal_lines
  ALTER COLUMN debit  TYPE NUMERIC(18,2) USING (debit  / 100.0),
  ALTER COLUMN credit TYPE NUMERIC(18,2) USING (credit / 100.0);

ALTER TABLE accounting.bank_statement_lines
  ALTER COLUMN debit  TYPE NUMERIC(18,2) USING (debit  / 100.0),
  ALTER COLUMN credit TYPE NUMERIC(18,2) USING (credit / 100.0);
