package payslips

import (
	"context"
	"errors"
	"fmt"

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

var ErrNotFound = errors.New("payslips: liquidación no encontrada")

const payslipCols = `id, company_id, employee_id, contract_id,
	period_year, period_month, worked_days, status,
	total_earned_cents, total_deducted_cents, net_pay_cents,
	journal_id, paid_at, created_at, updated_at`

const lineCols = `id, payslip_id, concept_code, concept_name, concept_type,
	quantity, amount_cents, created_at`

func (r *PostgresRepository) Create(ctx context.Context, in CreateInput, res Result) (*Payslip, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("payslips create: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO payroll.payslips
		 (company_id, employee_id, contract_id, period_year, period_month, worked_days,
		  total_earned_cents, total_deducted_cents, net_pay_cents)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING `+payslipCols,
		in.CompanyID, in.EmployeeID, in.ContractID,
		in.PeriodYear, in.PeriodMonth, in.WorkedDays,
		res.TotalEarnedCents, res.TotalDeductedCents, res.NetPayCents,
	)
	ps, err := scanPayslip(row)
	if err != nil {
		return nil, fmt.Errorf("payslips create: %w", err)
	}

	for _, l := range res.Lines {
		row := tx.QueryRow(ctx,
			`INSERT INTO payroll.payslip_lines
			 (payslip_id, concept_code, concept_name, concept_type, quantity, amount_cents)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 RETURNING `+lineCols,
			ps.ID, l.ConceptCode, l.ConceptName, l.ConceptType, l.Quantity, l.AmountCents,
		)
		line, err := scanLine(row)
		if err != nil {
			return nil, fmt.Errorf("payslips create line: %w", err)
		}
		ps.Lines = append(ps.Lines, line)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("payslips create: commit: %w", err)
	}
	return ps, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id uuid.UUID) (*Payslip, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+payslipCols+` FROM payroll.payslips WHERE id = $1`, id)
	ps, err := scanPayslip(row)
	if err != nil {
		return nil, fmt.Errorf("payslips get: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+lineCols+` FROM payroll.payslip_lines WHERE payslip_id = $1 ORDER BY created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("payslips get lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		l, err := scanLine(rows)
		if err != nil {
			return nil, err
		}
		ps.Lines = append(ps.Lines, l)
	}
	return ps, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context, companyID uuid.UUID, year, month int) ([]*Payslip, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+payslipCols+`
		 FROM payroll.payslips
		 WHERE company_id = $1 AND period_year = $2 AND period_month = $3
		 ORDER BY created_at`,
		companyID, year, month,
	)
	if err != nil {
		return nil, fmt.Errorf("payslips list: %w", err)
	}
	defer rows.Close()
	var out []*Payslip
	for rows.Next() {
		ps, err := scanPayslip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payroll.payslips SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("payslips update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetSMMLV(ctx context.Context, year int) (int64, error) {
	var cents int64
	err := r.pool.QueryRow(ctx,
		`SELECT amount_cents FROM payroll.smmlv_values WHERE year = $1`, year,
	).Scan(&cents)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("payslips: no hay SMMLV registrado para el año %d", year)
	}
	return cents, err
}

func (r *PostgresRepository) GetARLRate(ctx context.Context, year int, riskClass string) (int, error) {
	var rateBP int
	err := r.pool.QueryRow(ctx,
		`SELECT rate_bp FROM payroll.arl_rates WHERE year = $1 AND risk_class = $2`,
		year, riskClass,
	).Scan(&rateBP)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("payslips: no hay tasa ARL para año %d clase %s", year, riskClass)
	}
	return rateBP, err
}

func scanPayslip(row pgx.Row) (*Payslip, error) {
	var ps Payslip
	var journalID *uuid.UUID
	err := row.Scan(
		&ps.ID, &ps.CompanyID, &ps.EmployeeID, &ps.ContractID,
		&ps.PeriodYear, &ps.PeriodMonth, &ps.WorkedDays, &ps.Status,
		&ps.TotalEarnedCents, &ps.TotalDeductedCents, &ps.NetPayCents,
		&journalID, &ps.PaidAt, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan payslip: %w", err)
	}
	ps.JournalID = journalID
	return &ps, nil
}

func scanLine(row pgx.Row) (*PayslipLine, error) {
	var l PayslipLine
	err := row.Scan(
		&l.ID, &l.PayslipID,
		&l.ConceptCode, &l.ConceptName, &l.ConceptType,
		&l.Quantity, &l.AmountCents, &l.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan payslip line: %w", err)
	}
	return &l, nil
}
