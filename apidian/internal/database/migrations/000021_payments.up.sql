CREATE TABLE payments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id   UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    type        TEXT        NOT NULL CHECK (type IN ('affiliation', 'renewal', 'documents')),
    amount_cop  INT         NOT NULL,
    note        TEXT,
    paid_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX payments_issuer_id_idx ON payments (issuer_id);
CREATE INDEX payments_paid_at_idx   ON payments (paid_at DESC);
