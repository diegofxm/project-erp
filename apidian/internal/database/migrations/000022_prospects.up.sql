CREATE TABLE prospects (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    email       TEXT        NOT NULL UNIQUE,
    nit         TEXT,
    cedula_pdf  BYTEA,
    rut_pdf     BYTEA,
    status      TEXT        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
    notes       TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX prospects_status_idx      ON prospects (status);
CREATE INDEX prospects_created_at_idx  ON prospects (created_at DESC);
