package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, p Payment) (*Payment, error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	if p.PaidAt.IsZero() {
		p.PaidAt = now
	}
	p.CreatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO payments (id, issuer_id, type, amount_cop, note, paid_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.IssuerID, p.Type, p.AmountCOP, p.Note, p.PaidAt, p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]Payment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, issuer_id, type, amount_cop, note, paid_at, created_at
		FROM payments
		WHERE issuer_id = $1
		ORDER BY paid_at DESC`,
		issuerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var out []Payment
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.IssuerID, &p.Type, &p.AmountCOP, &p.Note, &p.PaidAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
