ALTER TABLE issuers
    DROP COLUMN IF EXISTS ne_software_id,
    DROP COLUMN IF EXISTS ne_software_pin;
