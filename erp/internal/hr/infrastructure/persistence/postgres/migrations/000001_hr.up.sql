CREATE SCHEMA IF NOT EXISTS hr;

-- absence_type: 'vacation' | 'sick_leave' | 'maternity' | 'paternity' | 'personal' | 'unpaid'
-- status: 'pending' | 'approved' | 'rejected'
-- days: calculado al crear (días calendario entre start_date y end_date inclusive)

CREATE TABLE IF NOT EXISTS hr.absences (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID        NOT NULL,
    employee_id UUID        NOT NULL,
    type        VARCHAR(20) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    start_date  DATE        NOT NULL,
    end_date    DATE        NOT NULL,
    days        SMALLINT    NOT NULL,
    reason      TEXT,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_absences_company    ON hr.absences(company_id, status);
CREATE INDEX IF NOT EXISTS idx_absences_employee   ON hr.absences(employee_id);
