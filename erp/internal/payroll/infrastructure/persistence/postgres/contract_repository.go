package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/payroll/domain"
)

type ContractRepository struct {
	pool *pgxpool.Pool
}

func NewContractRepository(pool *pgxpool.Pool) *ContractRepository {
	return &ContractRepository{pool: pool}
}

const contractCols = `id, employee_id, company_id,
	contract_type, work_schedule, position, cost_center,
	salary_cents, salary_type, risk_class,
	start_date, end_date, termination_date, termination_cause,
	health_entity, pension_entity, arl_entity, caja_entity,
	is_active, created_at, updated_at`

func (r *ContractRepository) Create(ctx context.Context, in domain.CreateContractInput) (*domain.Contract, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payroll.contracts
		 (employee_id, company_id, contract_type, work_schedule, position, cost_center,
		  salary_cents, salary_type, risk_class, start_date, end_date,
		  health_entity, pension_entity, arl_entity, caja_entity)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING `+contractCols,
		in.EmployeeID, in.CompanyID, in.ContractType, in.WorkSchedule, in.Position, nilStr(in.CostCenter),
		in.SalaryCents, in.SalaryType, in.RiskClass, in.StartDate, in.EndDate,
		nilStr(in.HealthEntity), nilStr(in.PensionEntity), nilStr(in.ARLEntity), nilStr(in.CajaEntity),
	)
	c, err := scanContract(row)
	if err != nil {
		return nil, fmt.Errorf("contracts create: %w", err)
	}
	return c, nil
}

func (r *ContractRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Contract, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+contractCols+` FROM payroll.contracts WHERE id = $1`, id)
	c, err := scanContract(row)
	if err != nil {
		return nil, fmt.Errorf("contracts get: %w", err)
	}
	return c, nil
}

func (r *ContractRepository) GetActive(ctx context.Context, employeeID uuid.UUID) (*domain.Contract, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+contractCols+`
		 FROM payroll.contracts
		 WHERE employee_id = $1 AND is_active = true
		 ORDER BY start_date DESC LIMIT 1`,
		employeeID,
	)
	c, err := scanContract(row)
	if err != nil {
		return nil, fmt.Errorf("contracts get active: %w", err)
	}
	return c, nil
}

func (r *ContractRepository) ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]*domain.Contract, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+contractCols+`
		 FROM payroll.contracts
		 WHERE employee_id = $1
		 ORDER BY start_date DESC`,
		employeeID,
	)
	if err != nil {
		return nil, fmt.Errorf("contracts list: %w", err)
	}
	defer rows.Close()

	var out []*domain.Contract
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ContractRepository) Terminate(ctx context.Context, id uuid.UUID, cause string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payroll.contracts
		 SET is_active = false, termination_date = NOW(), termination_cause = $2, updated_at = NOW()
		 WHERE id = $1`,
		id, cause,
	)
	if err != nil {
		return fmt.Errorf("contracts terminate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrContractNotFound
	}
	return nil
}

func scanContract(row pgx.Row) (*domain.Contract, error) {
	var c domain.Contract
	var costCenter, termCause, healthEnt, pensionEnt, arlEnt, cajaEnt *string
	var endDate, termDate *time.Time

	err := row.Scan(
		&c.ID, &c.EmployeeID, &c.CompanyID,
		&c.ContractType, &c.WorkSchedule, &c.Position, &costCenter,
		&c.SalaryCents, &c.SalaryType, &c.RiskClass,
		&c.StartDate, &endDate, &termDate, &termCause,
		&healthEnt, &pensionEnt, &arlEnt, &cajaEnt,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrContractNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan contract: %w", err)
	}
	derefStr(&c.CostCenter, costCenter)
	derefStr(&c.TerminationCause, termCause)
	derefStr(&c.HealthEntity, healthEnt)
	derefStr(&c.PensionEntity, pensionEnt)
	derefStr(&c.ARLEntity, arlEnt)
	derefStr(&c.CajaEntity, cajaEnt)
	c.EndDate = endDate
	c.TerminationDate = termDate
	return &c, nil
}
