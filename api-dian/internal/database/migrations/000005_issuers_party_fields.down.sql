ALTER TABLE issuers
    DROP COLUMN IF EXISTS merchant_registration_number,
    DROP COLUMN IF EXISTS liability_codes,
    DROP COLUMN IF EXISTS tax_scheme_name,
    DROP COLUMN IF EXISTS tax_scheme_code,
    DROP COLUMN IF EXISTS entity_type_code;
