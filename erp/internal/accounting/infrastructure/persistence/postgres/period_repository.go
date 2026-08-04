package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
	"github.com/google/uuid"
)

type PeriodRepository struct {
	pool *pgxpool.Pool
}

func NewPeriodRepository(pool *pgxpool.Pool) *PeriodRepository {
	return &PeriodRepository{pool: pool}
}

func (r *PeriodRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.AccountingPeriod, error) {
	const q = `
		SELECT id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at
		FROM accounting.accounting_periods WHERE id = $1 AND company_id = $2`
	row := r.pool.QueryRow(ctx, q, id, companyID)
	p, err := scanPeriod(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPeriodNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PeriodRepository) GetByYearMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*domain.AccountingPeriod, error) {
	const q = `
		SELECT id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at
		FROM accounting.accounting_periods WHERE company_id = $1 AND year = $2 AND month = $3`
	row := r.pool.QueryRow(ctx, q, companyID, year, month)
	p, err := scanPeriod(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPeriodNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *PeriodRepository) Create(ctx context.Context, p domain.AccountingPeriod) (*domain.AccountingPeriod, error) {
	const q = `
		INSERT INTO accounting.accounting_periods (company_id, year, month)
		VALUES ($1, $2, $3)
		RETURNING id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q, p.CompanyID, p.Year, p.Month)
	return scanPeriod(row)
}

func (r *PeriodRepository) Close(ctx context.Context, companyID, id uuid.UUID) error {
	const q = `
		UPDATE accounting.accounting_periods
		SET status = 'CLOSED', closed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND company_id = $2`
	_, err := r.pool.Exec(ctx, q, id, companyID)
	return err
}

func (r *PeriodRepository) Reopen(ctx context.Context, companyID, id uuid.UUID) error {
	const q = `
		UPDATE accounting.accounting_periods
		SET status = 'OPEN', closed_at = NULL, updated_at = NOW()
		WHERE id = $1 AND company_id = $2`
	_, err := r.pool.Exec(ctx, q, id, companyID)
	return err
}

func (r *PeriodRepository) CloseAllForYear(ctx context.Context, companyID uuid.UUID, year int) error {
	const q = `
		UPDATE accounting.accounting_periods
		SET status = 'CLOSED', closed_at = NOW(), updated_at = NOW()
		WHERE company_id = $1 AND year = $2 AND status = 'OPEN'`
	_, err := r.pool.Exec(ctx, q, companyID, year)
	return err
}

func (r *PeriodRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.AccountingPeriod, error) {
	const q = `
		SELECT id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at
		FROM accounting.accounting_periods
		WHERE company_id = $1 ORDER BY year DESC, month DESC`
	rows, err := r.pool.Query(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AccountingPeriod
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanPeriod(row pgx.Row) (*domain.AccountingPeriod, error) {
	var p domain.AccountingPeriod
	err := row.Scan(
		&p.ID, &p.CompanyID, &p.Year, &p.Month,
		&p.Status, &p.OpenedAt, &p.ClosedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
