package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/payroll/domain"
)

type PayslipRepository struct {
	pool *pgxpool.Pool
}

func NewPayslipRepository(pool *pgxpool.Pool) *PayslipRepository {
	return &PayslipRepository{pool: pool}
}

const payslipCols = `id, company_id, number, employee_id, contract_id,
	period_year, period_month, worked_days, status,
	total_earned_cents, total_deducted_cents, net_pay_cents,
	journal_id, paid_at, created_at, updated_at`

const lineCols = `id, payslip_id, concept_code, concept_name, concept_type,
	quantity, amount_cents, created_at`

func (r *PayslipRepository) Create(ctx context.Context, in domain.CreatePayslipInput, res domain.CalcResult) (*domain.Payslip, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("payslips create: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	seq, err := nextPayslipNumber(ctx, tx, in.CompanyID, in.PeriodYear)
	if err != nil {
		return nil, err
	}
	number := fmt.Sprintf("NOM-%d-%05d", in.PeriodYear, seq)

	row := tx.QueryRow(ctx,
		`INSERT INTO payroll.payslips
		 (company_id, number, employee_id, contract_id, period_year, period_month, worked_days,
		  total_earned_cents, total_deducted_cents, net_pay_cents)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING `+payslipCols,
		in.CompanyID, number, in.EmployeeID, in.ContractID,
		in.PeriodYear, in.PeriodMonth, in.WorkedDays,
		res.TotalEarnedCents, res.TotalDeductedCents, res.NetPayCents,
	)
	ps, err := scanPayslip(row)
	if err != nil {
		if isUniqueViol(err) {
			return nil, domain.ErrPayslipExists
		}
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

// querier abstrae *pgxpool.Pool/pgx.Tx — nextPayslipNumber corre dentro de la misma transacción
// que Create.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// nextPayslipNumber asigna el siguiente folio de desprendible para la empresa y el año dados —
// arranca en 1 cada año. Mismo patrón que sales.Repository.NextSaleNumber.
func nextPayslipNumber(ctx context.Context, q querier, companyID uuid.UUID, year int) (int, error) {
	const query = `
		INSERT INTO payroll.number_counters (company_id, year, last_seq)
		VALUES ($1, $2, 1)
		ON CONFLICT (company_id, year)
		DO UPDATE SET last_seq = payroll.number_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := q.QueryRow(ctx, query, companyID, year).Scan(&seq); err != nil {
		return 0, fmt.Errorf("asignar folio de desprendible: %w", err)
	}
	return seq, nil
}

func (r *PayslipRepository) NextPayslipNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error) {
	return nextPayslipNumber(ctx, r.pool, companyID, year)
}

func (r *PayslipRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payslip, error) {
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

func (r *PayslipRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, year, month int) ([]*domain.Payslip, error) {
	q := `SELECT ` + payslipCols + `
	      FROM payroll.payslips
	      WHERE company_id = $1`
	args := []any{companyID}
	if year > 0 {
		args = append(args, year)
		q += fmt.Sprintf(" AND period_year = $%d", len(args))
	}
	if month > 0 {
		args = append(args, month)
		q += fmt.Sprintf(" AND period_month = $%d", len(args))
	}
	q += " ORDER BY period_year DESC, period_month DESC, created_at DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("payslips list: %w", err)
	}
	defer rows.Close()
	var out []*domain.Payslip
	for rows.Next() {
		ps, err := scanPayslip(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ps)
	}
	return out, rows.Err()
}

func (r *PayslipRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PayslipStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE payroll.payslips SET status = $2, updated_at = NOW() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("payslips update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPayslipNotFound
	}
	return nil
}

func (r *PayslipRepository) GetSMMLV(ctx context.Context, year int) (int64, error) {
	var cents int64
	err := r.pool.QueryRow(ctx,
		`SELECT amount_cents FROM payroll.smmlv_values WHERE year = $1`, year,
	).Scan(&cents)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNoSMMLV
	}
	return cents, err
}

func (r *PayslipRepository) GetARLRate(ctx context.Context, year int, riskClass string) (int, error) {
	var rateBP int
	err := r.pool.QueryRow(ctx,
		`SELECT rate_bp FROM payroll.arl_rates WHERE year = $1 AND risk_class = $2`,
		year, riskClass,
	).Scan(&rateBP)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNoARLRate
	}
	return rateBP, err
}

func scanPayslip(row pgx.Row) (*domain.Payslip, error) {
	var ps domain.Payslip
	err := row.Scan(
		&ps.ID, &ps.CompanyID, &ps.Number, &ps.EmployeeID, &ps.ContractID,
		&ps.PeriodYear, &ps.PeriodMonth, &ps.WorkedDays, &ps.Status,
		&ps.TotalEarnedCents, &ps.TotalDeductedCents, &ps.NetPayCents,
		&ps.JournalID, &ps.PaidAt, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPayslipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan payslip: %w", err)
	}
	return &ps, nil
}

func scanLine(row pgx.Row) (*domain.PayslipLine, error) {
	var l domain.PayslipLine
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
