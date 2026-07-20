package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("subscription: not found")

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const selectJoin = `
	SELECT s.id, s.issuer_id, s.plan_id, p.name, p.max_documents_per_month,
	       s.status, s.started_at, s.ends_at, s.created_at, s.updated_at
	FROM subscriptions s
	JOIN plans p ON p.id = s.plan_id`

func scanSub(row pgx.Row) (*Subscription, error) {
	var s Subscription
	var statusStr string
	err := row.Scan(
		&s.ID, &s.IssuerID, &s.PlanID, &s.PlanName, &s.MaxDocumentsPerMonth,
		&statusStr, &s.StartedAt, &s.EndsAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	s.Status = Status(statusStr)
	return &s, nil
}

// GetActive devuelve la suscripción activa del emisor. Devuelve ErrNotFound si no tiene ninguna.
func (r *PostgresRepository) GetActive(ctx context.Context, issuerID uuid.UUID) (*Subscription, error) {
	return scanSub(r.pool.QueryRow(ctx, selectJoin+`
		WHERE s.issuer_id = $1 AND s.status = 'active'
	`, issuerID))
}

// ListByIssuer devuelve todas las suscripciones de un emisor (historial).
func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]Subscription, error) {
	rows, err := r.pool.Query(ctx, selectJoin+`
		WHERE s.issuer_id = $1 ORDER BY s.created_at DESC
	`, issuerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var s Subscription
		var statusStr string
		if err := rows.Scan(
			&s.ID, &s.IssuerID, &s.PlanID, &s.PlanName, &s.MaxDocumentsPerMonth,
			&statusStr, &s.StartedAt, &s.EndsAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		s.Status = Status(statusStr)
		out = append(out, s)
	}
	return out, rows.Err()
}

// Assign cancela la suscripción activa actual (si la hay) y crea una nueva activa con el plan dado.
func (r *PostgresRepository) Assign(ctx context.Context, issuerID, planID uuid.UUID) (*Subscription, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()

	// Cancelar suscripción activa actual
	_, err = tx.Exec(ctx, `
		UPDATE subscriptions SET status = 'cancelled', ends_at = $1, updated_at = $1
		WHERE issuer_id = $2 AND status = 'active'
	`, now, issuerID)
	if err != nil {
		return nil, err
	}

	// Crear nueva
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO subscriptions (id, issuer_id, plan_id, status, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'active', $4, $4, $4)
	`, id, issuerID, planID, now)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetActive(ctx, issuerID)
}

// CountDocumentsThisMonth cuenta los documentos emitidos (confirmados) por el emisor en el mes actual.
func (r *PostgresRepository) CountDocumentsThisMonth(ctx context.Context, issuerID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM documents
		WHERE issuer_id = $1
		  AND status NOT IN ('draft', 'rejected', 'send_error')
		  AND DATE_TRUNC('month', issue_date) = DATE_TRUNC('month', CURRENT_DATE)
	`, issuerID).Scan(&count)
	return count, err
}
