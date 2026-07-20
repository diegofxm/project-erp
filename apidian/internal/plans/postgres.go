package plans

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("plan: not found")

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func scanPlan(row pgx.Row) (*Plan, error) {
	var p Plan
	err := row.Scan(
		&p.ID, &p.Name, &p.Description,
		&p.MaxDocumentsPerMonth, &p.MaxIssuers, &p.PriceCOP,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Plan, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, max_documents_per_month, max_issuers, price_cop,
		       is_active, created_at, updated_at
		FROM plans ORDER BY price_cop ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Plan
	for rows.Next() {
		var p Plan
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description,
			&p.MaxDocumentsPerMonth, &p.MaxIssuers, &p.PriceCOP,
			&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Plan, error) {
	return scanPlan(r.pool.QueryRow(ctx, `
		SELECT id, name, description, max_documents_per_month, max_issuers, price_cop,
		       is_active, created_at, updated_at
		FROM plans WHERE id = $1
	`, id))
}

func (r *PostgresRepository) GetFree(ctx context.Context) (*Plan, error) {
	return scanPlan(r.pool.QueryRow(ctx, `
		SELECT id, name, description, max_documents_per_month, max_issuers, price_cop,
		       is_active, created_at, updated_at
		FROM plans WHERE price_cop = 0 AND is_active = TRUE
		ORDER BY created_at ASC LIMIT 1
	`))
}

func (r *PostgresRepository) Create(ctx context.Context, p Plan) (*Plan, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO plans (id, name, description, max_documents_per_month, max_issuers, price_cop, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, p.ID, p.Name, p.Description, p.MaxDocumentsPerMonth, p.MaxIssuers, p.PriceCOP, true, now, now)
	if err != nil {
		return nil, err
	}
	p.IsActive = true
	p.CreatedAt = now
	p.UpdatedAt = now
	return &p, nil
}

func (r *PostgresRepository) Update(ctx context.Context, p Plan) (*Plan, error) {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE plans SET
			name = $1, description = $2,
			max_documents_per_month = $3, max_issuers = $4,
			price_cop = $5, is_active = $6, updated_at = $7
		WHERE id = $8
	`, p.Name, p.Description, p.MaxDocumentsPerMonth, p.MaxIssuers, p.PriceCOP, p.IsActive, now, p.ID)
	if err != nil {
		return nil, err
	}
	p.UpdatedAt = now
	return &p, nil
}
