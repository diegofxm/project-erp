package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type SettingsRepository struct{ pool *pgxpool.Pool }

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	var s domain.Settings
	err := r.pool.QueryRow(ctx, "SELECT iva_rate_bp, updated_at FROM saas.settings WHERE id=1").
		Scan(&s.IVARateBP, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("obtener configuración: %w", err)
	}
	return &s, nil
}

func (r *SettingsRepository) Update(ctx context.Context, s domain.Settings) (*domain.Settings, error) {
	var out domain.Settings
	err := r.pool.QueryRow(ctx, `
		UPDATE saas.settings SET iva_rate_bp=$1, updated_at=NOW() WHERE id=1
		RETURNING iva_rate_bp, updated_at`,
		s.IVARateBP,
	).Scan(&out.IVARateBP, &out.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("actualizar configuración: %w", err)
	}
	return &out, nil
}
