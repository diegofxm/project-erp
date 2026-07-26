package periods

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

const periodCols = `id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at`

func (r *PostgresRepository) Create(ctx context.Context, p AccountingPeriod) (*AccountingPeriod, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.OpenedAt = now
	p.Status = StatusOpen

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.accounting_periods (id, company_id, year, month, status, opened_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.CompanyID, p.Year, p.Month, p.Status, p.OpenedAt, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create period: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*AccountingPeriod, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+periodCols+` FROM accounting.accounting_periods WHERE id = $1`, id)
	return scanPeriod(row)
}

func (r *PostgresRepository) GetByYearMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*AccountingPeriod, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+periodCols+` FROM accounting.accounting_periods WHERE company_id = $1 AND year = $2 AND month = $3`,
		companyID, year, month,
	)
	return scanPeriod(row)
}

func (r *PostgresRepository) List(ctx context.Context, companyID uuid.UUID) ([]*AccountingPeriod, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+periodCols+` FROM accounting.accounting_periods WHERE company_id = $1 ORDER BY year DESC, month DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("list periods: %w", err)
	}
	defer rows.Close()

	var out []*AccountingPeriod
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Close(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.accounting_periods
		SET status = $1, closed_at = $2, updated_at = $3
		WHERE id = $4 AND status = 'OPEN'`,
		StatusClosed, now, now, id,
	)
	if err != nil {
		return fmt.Errorf("close period: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPeriodNotFound
	}
	return nil
}

func (r *PostgresRepository) CloseAllForYear(ctx context.Context, companyID uuid.UUID, year int) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE accounting.accounting_periods
		SET status = $1, closed_at = $2, updated_at = $3
		WHERE company_id = $4 AND year = $5 AND status = 'OPEN'`,
		StatusClosed, now, now, companyID, year,
	)
	if err != nil {
		return fmt.Errorf("close all periods for year: %w", err)
	}
	return nil
}

func scanPeriod(row pgx.Row) (*AccountingPeriod, error) {
	var p AccountingPeriod
	var closedAt *time.Time

	err := row.Scan(
		&p.ID, &p.CompanyID, &p.Year, &p.Month,
		&p.Status, &p.OpenedAt, &closedAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPeriodNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan period: %w", err)
	}
	p.ClosedAt = closedAt
	return &p, nil
}
