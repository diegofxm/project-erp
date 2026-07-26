DROP INDEX IF EXISTS accounting.journal_lines_forex_idx;
ALTER TABLE accounting.journal_lines
    DROP COLUMN IF EXISTS foreign_currency,
    DROP COLUMN IF EXISTS foreign_amount;
DROP TABLE IF EXISTS accounting.exchange_rates;
