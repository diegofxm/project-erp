ALTER TABLE issuer_settings
    DROP COLUMN IF EXISTS renewal_due_at,
    DROP COLUMN IF EXISTS affiliated_at,
    DROP COLUMN IF EXISTS renewal_fee_cop,
    DROP COLUMN IF EXISTS affiliation_fee_cop;

ALTER TABLE plans
    DROP COLUMN IF EXISTS annual_increment_pct,
    DROP COLUMN IF EXISTS renewal_fee_cop,
    DROP COLUMN IF EXISTS affiliation_fee_cop;
