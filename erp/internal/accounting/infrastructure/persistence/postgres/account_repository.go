package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) GetByCode(ctx context.Context, code string) (*domain.Account, error) {
	const q = `
		SELECT id, code, name, COALESCE(parent_code,''), level, category, is_posting, is_active,
		       created_at, updated_at
		FROM accounting.accounts WHERE code = $1`
	row := r.pool.QueryRow(ctx, q, code)
	a, err := scanAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return a, nil
}

func (r *AccountRepository) GetPostable(ctx context.Context, code string) (*domain.Account, error) {
	a, err := r.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !a.IsActive {
		return nil, domain.ErrAccountInactive
	}
	if !a.IsPosting {
		return nil, domain.ErrAccountNotPosting
	}
	return a, nil
}

func (r *AccountRepository) List(ctx context.Context) ([]domain.Account, error) {
	const q = `
		SELECT id, code, name, COALESCE(parent_code,''), level, category, is_posting, is_active,
		       created_at, updated_at
		FROM accounting.accounts ORDER BY code`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func scanAccount(row pgx.Row) (*domain.Account, error) {
	var a domain.Account
	err := row.Scan(
		&a.ID, &a.Code, &a.Name, &a.ParentCode,
		&a.Level, &a.Category, &a.IsPosting, &a.IsActive,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}
