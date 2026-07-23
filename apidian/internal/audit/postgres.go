package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/google/uuid"
)

// PostgresRepository implementa Repository sobre pgxpool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository construye el repo.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Insert(ctx context.Context, e Event) error {
	var meta []byte
	if len(e.Metadata) > 0 {
		var err error
		meta, err = json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("audit: marshal metadata: %w", err)
		}
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_events (issuer_id, user_id, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		e.IssuerID, e.UserID, e.Action, e.ResourceType, e.ResourceID, meta,
	)
	return err
}

func (r *PostgresRepository) List(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]Event, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	args := []any{issuerID}
	query := `
		SELECT
			ae.id, ae.issuer_id, ae.user_id,
			COALESCE(u.name, '')  AS user_name,
			COALESCE(u.email, '') AS user_email,
			ae.action, ae.resource_type, ae.resource_id,
			ae.metadata, ae.created_at
		FROM audit_events ae
		LEFT JOIN users u ON u.id = ae.user_id
		WHERE ae.issuer_id = $1`

	if filter.ResourceID != nil {
		args = append(args, *filter.ResourceID)
		query += fmt.Sprintf(" AND ae.resource_id = $%d", len(args))
	}

	args = append(args, limit, filter.Offset)
	query += fmt.Sprintf(" ORDER BY ae.created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var metaRaw []byte
		if err := rows.Scan(
			&e.ID, &e.IssuerID, &e.UserID,
			&e.UserName, &e.UserEmail,
			&e.Action, &e.ResourceType, &e.ResourceID,
			&metaRaw, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
