package budget

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

func (r *PostgresRepository) CreateBudget(ctx context.Context, b Budget) (*Budget, error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	if b.Status == "" {
		b.Status = StatusDraft
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.budgets (id, company_id, year, name, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		b.ID, b.CompanyID, b.Year, b.Name, string(b.Status), b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create budget: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) GetBudget(ctx context.Context, id uuid.UUID) (*Budget, error) {
	var b Budget
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, year, name, status, created_at, updated_at
		FROM accounting.budgets WHERE id = $1`, id).
		Scan(&b.ID, &b.CompanyID, &b.Year, &b.Name, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBudgetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get budget: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) ListBudgets(ctx context.Context, companyID uuid.UUID, year int) ([]*Budget, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, year, name, status, created_at, updated_at
		FROM accounting.budgets
		WHERE company_id = $1 AND year = $2
		ORDER BY name`,
		companyID, year,
	)
	if err != nil {
		return nil, fmt.Errorf("list budgets: %w", err)
	}
	defer rows.Close()

	var out []*Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.CompanyID, &b.Year, &b.Name, &b.Status,
			&b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ApproveBudget(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounting.budgets
		SET status = 'APPROVED', updated_at = NOW()
		WHERE id = $1 AND status = 'DRAFT'`, id)
	if err != nil {
		return fmt.Errorf("approve budget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBudgetNotFound
	}
	return nil
}

func (r *PostgresRepository) SetLine(ctx context.Context, accountID uuid.UUID, req SetLineRequest) (*BudgetLine, error) {
	lineID := uuid.New()
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO accounting.budget_lines
			(id, budget_id, account_id,
			 jan, feb, mar, apr, may, jun,
			 jul, aug, sep, oct, nov, dec,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (budget_id, account_id) DO UPDATE SET
			jan=$4, feb=$5, mar=$6, apr=$7, may=$8, jun=$9,
			jul=$10, aug=$11, sep=$12, oct=$13, nov=$14, dec=$15,
			updated_at=NOW()`,
		lineID, req.BudgetID, accountID,
		req.Jan, req.Feb, req.Mar, req.Apr, req.May, req.Jun,
		req.Jul, req.Aug, req.Sep, req.Oct, req.Nov, req.Dec,
		now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("set budget line: %w", err)
	}

	var l BudgetLine
	err = r.pool.QueryRow(ctx, `
		SELECT bl.id, bl.budget_id, bl.account_id, a.code, a.name, a.category,
		       bl.jan, bl.feb, bl.mar, bl.apr, bl.may, bl.jun,
		       bl.jul, bl.aug, bl.sep, bl.oct, bl.nov, bl.dec,
		       bl.created_at, bl.updated_at
		FROM accounting.budget_lines bl
		JOIN accounting.accounts a ON a.id = bl.account_id
		WHERE bl.budget_id = $1 AND bl.account_id = $2`,
		req.BudgetID, accountID,
	).Scan(
		&l.ID, &l.BudgetID, &l.AccountID, &l.AccountCode, &l.AccountName, &l.Category,
		&l.Jan, &l.Feb, &l.Mar, &l.Apr, &l.May, &l.Jun,
		&l.Jul, &l.Aug, &l.Sep, &l.Oct, &l.Nov, &l.Dec,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("re-read budget line: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) Lines(ctx context.Context, budgetID uuid.UUID) ([]*BudgetLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bl.id, bl.budget_id, bl.account_id, a.code, a.name, a.category,
		       bl.jan, bl.feb, bl.mar, bl.apr, bl.may, bl.jun,
		       bl.jul, bl.aug, bl.sep, bl.oct, bl.nov, bl.dec,
		       bl.created_at, bl.updated_at
		FROM accounting.budget_lines bl
		JOIN accounting.accounts a ON a.id = bl.account_id
		WHERE bl.budget_id = $1
		ORDER BY a.code`,
		budgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("list budget lines: %w", err)
	}
	defer rows.Close()

	var out []*BudgetLine
	for rows.Next() {
		var l BudgetLine
		if err := rows.Scan(
			&l.ID, &l.BudgetID, &l.AccountID, &l.AccountCode, &l.AccountName, &l.Category,
			&l.Jan, &l.Feb, &l.Mar, &l.Apr, &l.May, &l.Jun,
			&l.Jul, &l.Aug, &l.Sep, &l.Oct, &l.Nov, &l.Dec,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan budget line: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ActualsByMonth(ctx context.Context, companyID uuid.UUID, year, fromMonth, toMonth int) ([]ActualRow, error) {
	from := time.Date(year, time.Month(fromMonth), 1, 0, 0, 0, 0, time.UTC)
	var to time.Time
	if toMonth == 12 {
		to = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		to = time.Date(year, time.Month(toMonth+1), 1, 0, 0, 0, 0, time.UTC)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT jl.account_id, a.code, a.name, a.category,
		       COALESCE(SUM(jl.debit) - SUM(jl.credit), 0) AS net
		FROM accounting.journal_lines jl
		JOIN accounting.journal_entries je ON je.id = jl.journal_id
		JOIN accounting.accounts a ON a.id = jl.account_id
		WHERE je.company_id = $1
		  AND je.status = 'POSTED'
		  AND je.date >= $2
		  AND je.date <  $3
		GROUP BY jl.account_id, a.code, a.name, a.category
		ORDER BY a.code`,
		companyID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("actuals by month: %w", err)
	}
	defer rows.Close()

	var out []ActualRow
	for rows.Next() {
		var a ActualRow
		if err := rows.Scan(&a.AccountID, &a.AccountCode, &a.AccountName, &a.Category, &a.Net); err != nil {
			return nil, fmt.Errorf("scan actuals: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
