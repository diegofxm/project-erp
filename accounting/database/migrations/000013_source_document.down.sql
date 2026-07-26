DROP INDEX IF EXISTS accounting.fixed_assets_source_doc_idx;
ALTER TABLE accounting.fixed_assets
    DROP COLUMN IF EXISTS source_document_type,
    DROP COLUMN IF EXISTS source_document_id;

DROP INDEX IF EXISTS accounting.journal_entries_source_doc_idx;
ALTER TABLE accounting.journal_entries
    DROP COLUMN IF EXISTS source_document_type,
    DROP COLUMN IF EXISTS source_document_id;
