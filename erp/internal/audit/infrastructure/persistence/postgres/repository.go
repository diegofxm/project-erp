package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/audit/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any) error {
	var metaBytes []byte
	if metadata != nil {
		var err error
		metaBytes, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("serializar metadata: %w", err)
		}
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit.events (company_id, user_id, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		companyID, userID, action, resourceType, resourceID, metaBytes,
	)
	if err != nil {
		return fmt.Errorf("insertar audit event: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Event, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var rows pgx.Rows
	var err error
	if filter.ResourceID != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT ae.id, ae.company_id, ae.user_id, ae.action, ae.resource_type, ae.resource_id, ae.metadata, ae.created_at,
			       COALESCE(u.email, '') AS user_email, COALESCE(u.name, '') AS user_name
			FROM audit.events ae
			LEFT JOIN security.users u ON u.id = ae.user_id
			WHERE ae.company_id = $1 AND ae.resource_id = $2
			ORDER BY ae.created_at DESC
			LIMIT $3 OFFSET $4`,
			companyID, filter.ResourceID, limit, filter.Offset,
		)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT ae.id, ae.company_id, ae.user_id, ae.action, ae.resource_type, ae.resource_id, ae.metadata, ae.created_at,
			       COALESCE(u.email, '') AS user_email, COALESCE(u.name, '') AS user_name
			FROM audit.events ae
			LEFT JOIN security.users u ON u.id = ae.user_id
			WHERE ae.company_id = $1
			ORDER BY ae.created_at DESC
			LIMIT $2 OFFSET $3`,
			companyID, limit, filter.Offset,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listar audit events: %w", err)
	}
	defer rows.Close()

	var out []*domain.Event
	for rows.Next() {
		var e domain.Event
		var metaBytes []byte
		if err := rows.Scan(
			&e.ID, &e.CompanyID, &e.UserID, &e.Action, &e.ResourceType, &e.ResourceID, &metaBytes, &e.CreatedAt,
			&e.UserEmail, &e.UserName,
		); err != nil {
			return nil, fmt.Errorf("leer audit event: %w", err)
		}
		if metaBytes != nil {
			_ = json.Unmarshal(metaBytes, &e.Metadata)
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}
