package accounts

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

const accountCols = `id, code, name, parent_code, level, category, is_posting, is_active, created_at, updated_at`

func (r *PostgresRepository) GetByCode(ctx context.Context, code string) (*Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounting.accounts WHERE code = $1`, code)
	return scanAccount(row)
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*Account, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounting.accounts WHERE id = $1`, id)
	return scanAccount(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]*Account, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+accountCols+` FROM accounting.accounts WHERE is_active = true ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var out []*Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAccount(row pgx.Row) (*Account, error) {
	var a Account
	var parentCode *string
	var id uuid.UUID

	err := row.Scan(
		&id, &a.Code, &a.Name, &parentCode,
		&a.Level, &a.Category, &a.IsPosting, &a.IsActive,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan account: %w", err)
	}
	a.ID = id
	if parentCode != nil {
		a.ParentCode = *parentCode
	}
	_ = time.UTC
	return &a, nil
}
