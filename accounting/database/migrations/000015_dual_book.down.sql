DROP INDEX IF EXISTS accounting.journal_entries_book_idx;
ALTER TABLE accounting.journal_entries
    DROP CONSTRAINT IF EXISTS journal_entries_book_check,
    DROP COLUMN IF EXISTS book;
