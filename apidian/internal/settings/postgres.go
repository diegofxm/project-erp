package settings

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Get devuelve la configuración del emisor. Si no existe devuelve los valores por defecto
// sin insertar nada — el upsert solo ocurre al guardar.
func (r *PostgresRepository) Get(ctx context.Context, issuerID uuid.UUID) (*IssuerSettings, error) {
	var s IssuerSettings
	err := r.pool.QueryRow(ctx, `
		SELECT issuer_id, brand_color, updated_at
		FROM issuer_settings WHERE issuer_id = $1
	`, issuerID).Scan(&s.IssuerID, &s.BrandColor, &s.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return &IssuerSettings{
			IssuerID:   issuerID,
			BrandColor: DefaultBrandColor,
			UpdatedAt:  time.Now().UTC(),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Save hace upsert de la configuración del emisor.
func (r *PostgresRepository) Save(ctx context.Context, s IssuerSettings) (*IssuerSettings, error) {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO issuer_settings (issuer_id, brand_color, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (issuer_id) DO UPDATE SET
			brand_color = EXCLUDED.brand_color,
			updated_at  = EXCLUDED.updated_at
	`, s.IssuerID, s.BrandColor, now)
	if err != nil {
		return nil, err
	}
	s.UpdatedAt = now
	return &s, nil
}
