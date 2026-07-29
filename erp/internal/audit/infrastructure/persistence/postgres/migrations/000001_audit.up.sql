CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID NOT NULL,
    user_id       UUID,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL DEFAULT '',
    resource_id   UUID,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_company_ts ON audit.events(company_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource   ON audit.events(resource_id) WHERE resource_id IS NOT NULL;
