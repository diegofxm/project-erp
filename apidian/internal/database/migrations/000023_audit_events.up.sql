CREATE TABLE audit_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer_id     UUID        NOT NULL REFERENCES issuers(id) ON DELETE CASCADE,
    user_id       UUID        REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT        NOT NULL,
    resource_type TEXT,
    resource_id   UUID,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_events_issuer_created_idx  ON audit_events (issuer_id, created_at DESC);
CREATE INDEX audit_events_resource_id_idx     ON audit_events (resource_id) WHERE resource_id IS NOT NULL;
