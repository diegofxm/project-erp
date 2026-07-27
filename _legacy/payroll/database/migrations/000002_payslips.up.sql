-- Liquidaciones de nómina.
-- concept_type en payslip_lines: 'earned' | 'deduction' | 'employer_contribution'
--   earned: devengados que recibe el empleado (salario, auxilio, horas extra, primas...).
--   deduction: descuentos al neto del empleado (salud empleado, pensión empleado, retención...).
--   employer_contribution: aportes parafiscales del empleador (no afectan neto del empleado).
--
-- journal_id apunta a accounting.journal_entries cuando la nómina se contabiliza.
-- Es solo un UUID — no hay FK entre esquemas para mantener independencia de módulos.

CREATE TABLE payroll.payslips (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID        NOT NULL,
    employee_id           UUID        NOT NULL REFERENCES payroll.employees(id),
    contract_id           UUID        NOT NULL REFERENCES payroll.contracts(id),
    period_year           SMALLINT    NOT NULL,
    period_month          SMALLINT    NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    worked_days           SMALLINT    NOT NULL DEFAULT 30,
    status                VARCHAR(20) NOT NULL DEFAULT 'draft',
    total_earned_cents    BIGINT      NOT NULL DEFAULT 0,
    total_deducted_cents  BIGINT      NOT NULL DEFAULT 0,
    net_pay_cents         BIGINT      NOT NULL DEFAULT 0,
    journal_id            UUID,
    paid_at               TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, employee_id, period_year, period_month)
);

CREATE INDEX idx_payslips_company_period
    ON payroll.payslips(company_id, period_year, period_month);
CREATE INDEX idx_payslips_employee
    ON payroll.payslips(employee_id);

-- Líneas de la liquidación.
-- quantity: días trabajados, horas extra, etc. (default 1 para conceptos de monto fijo).
CREATE TABLE payroll.payslip_lines (
    id            UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    payslip_id    UUID           NOT NULL REFERENCES payroll.payslips(id) ON DELETE CASCADE,
    concept_code  VARCHAR(20)    NOT NULL,
    concept_name  TEXT           NOT NULL,
    concept_type  VARCHAR(30)    NOT NULL,
    quantity      NUMERIC(10,4)  NOT NULL DEFAULT 1,
    amount_cents  BIGINT         NOT NULL,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payslip_lines_payslip ON payroll.payslip_lines(payslip_id);
