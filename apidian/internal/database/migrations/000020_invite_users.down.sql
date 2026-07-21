ALTER TABLE users
    DROP COLUMN IF EXISTS invite_accepted_at,
    DROP COLUMN IF EXISTS invite_token_expires_at,
    DROP COLUMN IF EXISTS invite_token,
    ALTER COLUMN password_hash SET NOT NULL;
