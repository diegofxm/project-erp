package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type BudgetRepository struct{ pool *pgxpool.Pool }

func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{pool: pool}
}

func (r *BudgetRepository) Create(ctx context.Context, b domain.Budget) (*domain.Budget, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.budgets (company_id, year, name, status)
		VALUES ($1,$2,$3,'DRAFT')
		RETURNING id, company_id, year, name, status, created_at, updated_at`,
		b.CompanyID, b.Year, b.Name,
	)
	return scanBudget(row)
}

func (r *BudgetRepository) List(ctx context.Context, companyID uuid.UUID, year int) ([]domain.Budget, error) {
	var rows pgx.Rows
	var err error
	if year > 0 {
		rows, err = r.pool.Query(ctx, `SELECT id, company_id, year, name, status, created_at, updated_at
			FROM accounting.budgets WHERE company_id=$1 AND year=$2 ORDER BY year DESC, name`, companyID, year)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT id, company_id, year, name, status, created_at, updated_at
			FROM accounting.budgets WHERE company_id=$1 ORDER BY year DESC, name`, companyID)
	}
	if err != nil {
		return nil, fmt.Errorf("listar presupuestos: %w", err)
	}
	defer rows.Close()

	var out []domain.Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (r *BudgetRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Budget, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, company_id, year, name, status, created_at, updated_at
		FROM accounting.budgets WHERE id=$1 AND company_id=$2`, id, companyID)
	b, err := scanBudget(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrBudgetNotFound
	}
	return b, err
}

func (r *BudgetRepository) Rename(ctx context.Context, companyID, id uuid.UUID, name string) (*domain.Budget, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE accounting.budgets SET name=$3, updated_at=NOW()
		WHERE id=$1 AND company_id=$2 AND status='DRAFT'
		RETURNING id, company_id, year, name, status, created_at, updated_at`,
		id, companyID, name,
	)
	b, err := scanBudget(row)
	if errors.Is(err, pgx.ErrNoRows) {
		// Puede ser que no exista o que no esté en DRAFT -- distinguir para dar el error correcto.
		if _, getErr := r.GetByID(ctx, companyID, id); getErr != nil {
			return nil, getErr
		}
		return nil, domain.ErrBudgetNotDraft
	}
	return b, err
}

func (r *BudgetRepository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status FROM accounting.budgets WHERE id=$1 AND company_id=$2`, id, companyID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrBudgetNotFound
	}
	if err != nil {
		return fmt.Errorf("consultar presupuesto: %w", err)
	}
	if status != string(domain.BudgetDraft) {
		return domain.ErrBudgetNotDraft
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM accounting.budget_lines WHERE budget_id=$1`, id); err != nil {
		return fmt.Errorf("eliminar líneas: %w", err)
	}
	if _, err := r.pool.Exec(ctx, `DELETE FROM accounting.budgets WHERE id=$1 AND company_id=$2`, id, companyID); err != nil {
		return fmt.Errorf("eliminar presupuesto: %w", err)
	}
	return nil
}

func (r *BudgetRepository) DeleteLine(ctx context.Context, budgetID, accountID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM accounting.budget_lines WHERE budget_id=$1 AND account_id=$2`, budgetID, accountID)
	if err != nil {
		return fmt.Errorf("eliminar línea de presupuesto: %w", err)
	}
	return nil
}

func (r *BudgetRepository) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.BudgetStatus) error {
	_, err := r.pool.Exec(ctx, "UPDATE accounting.budgets SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3", string(status), id, companyID)
	return err
}

func scanBudget(row pgx.Row) (*domain.Budget, error) {
	var b domain.Budget
	var status string
	err := row.Scan(&b.ID, &b.CompanyID, &b.Year, &b.Name, &status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	b.Status = domain.BudgetStatus(status)
	return &b, nil
}

func (r *BudgetRepository) UpsertLine(ctx context.Context, l domain.BudgetLine) (*domain.BudgetLine, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO accounting.budget_lines (budget_id, account_id, jan, feb, mar, apr, may, jun, jul, aug, sep, oct, nov, dec)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (budget_id, account_id) DO UPDATE SET
			jan=EXCLUDED.jan, feb=EXCLUDED.feb, mar=EXCLUDED.mar, apr=EXCLUDED.apr,
			may=EXCLUDED.may, jun=EXCLUDED.jun, jul=EXCLUDED.jul, aug=EXCLUDED.aug,
			sep=EXCLUDED.sep, oct=EXCLUDED.oct, nov=EXCLUDED.nov, dec=EXCLUDED.dec, updated_at=NOW()
		RETURNING id, budget_id, account_id, jan, feb, mar, apr, may, jun, jul, aug, sep, oct, nov, dec`,
		l.BudgetID, l.AccountID,
		l.Months[0], l.Months[1], l.Months[2], l.Months[3], l.Months[4], l.Months[5],
		l.Months[6], l.Months[7], l.Months[8], l.Months[9], l.Months[10], l.Months[11],
	)
	var out domain.BudgetLine
	err := row.Scan(&out.ID, &out.BudgetID, &out.AccountID,
		&out.Months[0], &out.Months[1], &out.Months[2], &out.Months[3], &out.Months[4], &out.Months[5],
		&out.Months[6], &out.Months[7], &out.Months[8], &out.Months[9], &out.Months[10], &out.Months[11])
	if err != nil {
		return nil, fmt.Errorf("guardar línea de presupuesto: %w", err)
	}
	return &out, nil
}

func (r *BudgetRepository) ListLines(ctx context.Context, budgetID uuid.UUID) ([]domain.BudgetLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bl.id, bl.budget_id, bl.account_id, a.code, a.name,
		       bl.jan, bl.feb, bl.mar, bl.apr, bl.may, bl.jun, bl.jul, bl.aug, bl.sep, bl.oct, bl.nov, bl.dec
		FROM accounting.budget_lines bl
		JOIN accounting.accounts a ON a.id = bl.account_id
		WHERE bl.budget_id=$1 ORDER BY a.code`,
		budgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar líneas de presupuesto: %w", err)
	}
	defer rows.Close()

	var out []domain.BudgetLine
	for rows.Next() {
		var l domain.BudgetLine
		if err := rows.Scan(&l.ID, &l.BudgetID, &l.AccountID, &l.AccountCode, &l.AccountName,
			&l.Months[0], &l.Months[1], &l.Months[2], &l.Months[3], &l.Months[4], &l.Months[5],
			&l.Months[6], &l.Months[7], &l.Months[8], &l.Months[9], &l.Months[10], &l.Months[11]); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *BudgetRepository) GetActualMonths(ctx context.Context, companyID, accountID uuid.UUID, year int) ([12]int64, error) {
	var out [12]int64
	rows, err := r.pool.Query(ctx, `
		SELECT EXTRACT(MONTH FROM e.date)::int AS m, SUM(l.debit - l.credit)
		FROM accounting.journal_lines l
		JOIN accounting.journal_entries e ON e.id = l.journal_id
		WHERE e.company_id = $1 AND e.status = 'POSTED' AND l.account_id = $2
		  AND EXTRACT(YEAR FROM e.date) = $3
		GROUP BY m`,
		companyID, accountID, year,
	)
	if err != nil {
		return out, fmt.Errorf("movimiento real por mes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var month int
		var amount int64
		if err := rows.Scan(&month, &amount); err != nil {
			return out, err
		}
		if month >= 1 && month <= 12 {
			out[month-1] = amount
		}
	}
	return out, rows.Err()
}
