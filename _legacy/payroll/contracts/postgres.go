package contracts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const contractCols = `id, employee_id, company_id,
	contract_type, work_schedule, position, cost_center,
	salary_cents, salary_type, risk_class,
	start_date, end_date, termination_date, termination_cause,
	health_entity, pension_entity, arl_entity, caja_entity,
	is_active, created_at, updated_at`

func (r *PostgresRepository) Create(ctx context.Context, in CreateInput) (*Contract, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO payroll.contracts
		 (employee_id, company_id, contract_type, work_schedule, position, cost_center,
		  salary_cents, salary_type, risk_class, start_date, end_date,
		  health_entity, pension_entity, arl_entity, caja_entity)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING `+contractCols,
		in.EmployeeID, in.CompanyID, in.ContractType, in.WorkSchedule, in.Position, in.CostCenter,
		in.SalaryCents, in.SalaryType, in.RiskClass, in.StartDate, in.EndDate,
		in.HealthEntity, in.PensionEntity, in.ARLEntity, in.CajaEntity,
	)
	c, err := scanContract(row)
	if err != nil {
		return nil, fmt.Errorf("contracts create: %w", err)
	}
	return c, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (*Contract, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+contractCols+` FROM payroll.contracts WHERE id = $1`, id)
	c, err := scanContract(row)
	if err != nil {
		return nil, fmt.Errorf("contracts get: %w", err)
	}
	return c, nil
}

func (r *PostgresRepository) GetActive(ctx context.Context, employeeID uuid.UUID) (*Contract, error) {
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

func (r *PostgresRepository) List(ctx context.Context, companyID uuid.UUID) ([]*Contract, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+contractCols+`
		 FROM payroll.contracts
		 WHERE company_id = $1
		 ORDER BY start_date DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("contracts list: %w", err)
	}
	defer rows.Close()

	var out []*Contract
	for rows.Next() {
		c, err := scanContract(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Terminate(ctx context.Context, id uuid.UUID, cause string) error {
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
		return ErrNotFound
	}
	return nil
}

func scanContract(row pgx.Row) (*Contract, error) {
	var c Contract
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
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan contract: %w", err)
	}
	if costCenter != nil {
		c.CostCenter = *costCenter
	}
	if termCause != nil {
		c.TerminationCause = *termCause
	}
	if healthEnt != nil {
		c.HealthEntity = *healthEnt
	}
	if pensionEnt != nil {
		c.PensionEntity = *pensionEnt
	}
	if arlEnt != nil {
		c.ARLEntity = *arlEnt
	}
	if cajaEnt != nil {
		c.CajaEntity = *cajaEnt
	}
	c.EndDate = endDate
	c.TerminationDate = termDate
	return &c, nil
}
