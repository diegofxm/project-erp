ALTER TABLE users
    ALTER COLUMN password_hash DROP NOT NULL,
    ADD COLUMN invite_token            UUID UNIQUE,
    ADD COLUMN invite_token_expires_at TIMESTAMPTZ,
    ADD COLUMN invite_accepted_at      TIMESTAMPTZ;
