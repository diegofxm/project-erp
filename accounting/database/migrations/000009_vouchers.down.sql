DROP INDEX IF EXISTS accounting.journal_entries_voucher_number_idx;
ALTER TABLE accounting.journal_entries
    DROP COLUMN IF EXISTS voucher_number,
    DROP COLUMN IF EXISTS voucher_type;
DROP TABLE IF EXISTS accounting.voucher_counters;
DROP TABLE IF EXISTS accounting.voucher_types;
