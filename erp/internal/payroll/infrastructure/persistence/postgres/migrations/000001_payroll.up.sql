CREATE SCHEMA IF NOT EXISTS payroll;

CREATE TABLE IF NOT EXISTS payroll.smmlv_values (
    year         SMALLINT PRIMARY KEY,
    amount_cents BIGINT   NOT NULL
);

CREATE TABLE IF NOT EXISTS payroll.arl_rates (
    year       SMALLINT   NOT NULL,
    risk_class VARCHAR(3) NOT NULL,
    rate_bp    INT        NOT NULL,
    PRIMARY KEY (year, risk_class)
);

CREATE TABLE IF NOT EXISTS payroll.employees (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id               UUID        NOT NULL,
    identification_type_code VARCHAR(3)  NOT NULL,
    identification_number    TEXT        NOT NULL,
    first_name               TEXT        NOT NULL,
    last_name                TEXT        NOT NULL,
    email                    TEXT,
    phone                    TEXT,
    department_code          VARCHAR(2),
    municipality_code        VARCHAR(5),
    address_line             TEXT,
    is_active                BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_company_id_number
    ON payroll.employees(company_id, identification_number);
CREATE INDEX IF NOT EXISTS idx_employees_company ON payroll.employees(company_id);

CREATE TABLE IF NOT EXISTS payroll.contracts (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id       UUID        NOT NULL REFERENCES payroll.employees(id),
    company_id        UUID        NOT NULL,
    contract_type     VARCHAR(20) NOT NULL,
    work_schedule     VARCHAR(20) NOT NULL DEFAULT 'full_time',
    position          TEXT        NOT NULL,
    cost_center       TEXT,
    salary_cents      BIGINT      NOT NULL,
    salary_type       VARCHAR(20) NOT NULL DEFAULT 'ordinary',
    risk_class        VARCHAR(3)  NOT NULL DEFAULT 'I',
    start_date        DATE        NOT NULL,
    end_date          DATE,
    termination_date  DATE,
    termination_cause TEXT,
    health_entity     TEXT,
    pension_entity    TEXT,
    arl_entity        TEXT,
    caja_entity       TEXT,
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contracts_employee ON payroll.contracts(employee_id);
CREATE INDEX IF NOT EXISTS idx_contracts_company  ON payroll.contracts(company_id, is_active);

CREATE TABLE IF NOT EXISTS payroll.payslips (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID        NOT NULL,
    employee_id          UUID        NOT NULL REFERENCES payroll.employees(id),
    contract_id          UUID        NOT NULL REFERENCES payroll.contracts(id),
    period_year          SMALLINT    NOT NULL,
    period_month         SMALLINT    NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    worked_days          SMALLINT    NOT NULL DEFAULT 30,
    status               VARCHAR(20) NOT NULL DEFAULT 'draft',
    total_earned_cents   BIGINT      NOT NULL DEFAULT 0,
    total_deducted_cents BIGINT      NOT NULL DEFAULT 0,
    net_pay_cents        BIGINT      NOT NULL DEFAULT 0,
    journal_id           UUID,
    paid_at              TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, employee_id, period_year, period_month)
);

CREATE INDEX IF NOT EXISTS idx_payslips_company_period
    ON payroll.payslips(company_id, period_year, period_month);
CREATE INDEX IF NOT EXISTS idx_payslips_employee ON payroll.payslips(employee_id);

CREATE TABLE IF NOT EXISTS payroll.payslip_lines (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    payslip_id   UUID          NOT NULL REFERENCES payroll.payslips(id) ON DELETE CASCADE,
    concept_code VARCHAR(20)   NOT NULL,
    concept_name TEXT          NOT NULL,
    concept_type VARCHAR(30)   NOT NULL,
    quantity     NUMERIC(10,4) NOT NULL DEFAULT 1,
    amount_cents BIGINT        NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payslip_lines_payslip ON payroll.payslip_lines(payslip_id);
