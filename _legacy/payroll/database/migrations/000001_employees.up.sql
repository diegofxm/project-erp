-- Schema payroll: gestión de nómina colombiana.
-- Las referencias a catalogs.* son SQL-only; el módulo payroll no importa código de
-- otros módulos Go. Las referencias a accounting.journal_entries son solo por valor
-- (UUID del asiento contable generado al aprobar la nómina).

CREATE SCHEMA IF NOT EXISTS payroll;

-- ── Catálogos de ley ─────────────────────────────────────────────────────────────────────────

-- SMMLV vigente por año fiscal.
CREATE TABLE payroll.smmlv_values (
    year         SMALLINT PRIMARY KEY,
    amount_cents BIGINT   NOT NULL
);

-- Tasas ARL por clase de riesgo y año.
-- risk_class sigue la nomenclatura SURA/gobierno: 'I', 'II', 'III', 'IV', 'V'.
-- rate_bp en puntos base (1 bp = 0.01%), idéntico a la convención de accounting.
CREATE TABLE payroll.arl_rates (
    year       SMALLINT    NOT NULL,
    risk_class VARCHAR(3)  NOT NULL,
    rate_bp    INT         NOT NULL,
    PRIMARY KEY (year, risk_class)
);

-- ── Empleados ────────────────────────────────────────────────────────────────────────────────

CREATE TABLE payroll.employees (
    id                        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id                UUID        NOT NULL,
    identification_type_code  VARCHAR(3)  NOT NULL,
    identification_number     TEXT        NOT NULL,
    first_name                TEXT        NOT NULL,
    last_name                 TEXT        NOT NULL,
    email                     TEXT,
    phone                     TEXT,
    department_code           VARCHAR(2),
    municipality_code         VARCHAR(5),
    address_line              TEXT,
    is_active                 BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_employees_company_id_number
    ON payroll.employees(company_id, identification_number);
CREATE INDEX idx_employees_company ON payroll.employees(company_id);

-- ── Contratos laborales ──────────────────────────────────────────────────────────────────────
-- Un empleado puede tener varios contratos históricos; solo uno activo a la vez.
-- salary_type: 'ordinary' | 'integral'
--   ordinary: aplican auxilios y aportes normales.
--   integral: el salario ya incluye todas las prestaciones (mínimo 10 SMMLV).
-- risk_class: clase de riesgo ARL asignada al cargo ('I'..'V').

CREATE TABLE payroll.contracts (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id        UUID        NOT NULL REFERENCES payroll.employees(id),
    company_id         UUID        NOT NULL,
    contract_type      VARCHAR(20) NOT NULL,  -- 'fijo' | 'indefinido' | 'obra' | 'servicios'
    work_schedule      VARCHAR(20) NOT NULL DEFAULT 'full_time',
    position           TEXT        NOT NULL,
    cost_center        TEXT,
    salary_cents       BIGINT      NOT NULL,
    salary_type        VARCHAR(20) NOT NULL DEFAULT 'ordinary',
    risk_class         VARCHAR(3)  NOT NULL DEFAULT 'I',
    start_date         DATE        NOT NULL,
    end_date           DATE,
    termination_date   DATE,
    termination_cause  TEXT,
    health_entity      TEXT,
    pension_entity     TEXT,
    arl_entity         TEXT,
    caja_entity        TEXT,
    is_active          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contracts_employee ON payroll.contracts(employee_id);
CREATE INDEX idx_contracts_company  ON payroll.contracts(company_id, is_active);
