ALTER TABLE documents
    DROP COLUMN IF EXISTS vendor,
    DROP COLUMN IF EXISTS operation_type_code,
    DROP COLUMN IF EXISTS withholding_taxes;
