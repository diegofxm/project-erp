package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type ModuleRepository struct{ pool *pgxpool.Pool }

func NewModuleRepository(pool *pgxpool.Pool) *ModuleRepository {
	return &ModuleRepository{pool: pool}
}

const moduleCols = "id, code, name, description"

func scanModule(row pgx.Row) (*domain.Module, error) {
	var m domain.Module
	if err := row.Scan(&m.ID, &m.Code, &m.Name, &m.Description); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *ModuleRepository) List(ctx context.Context) ([]domain.Module, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+moduleCols+" FROM saas.modules ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("listar módulos: %w", err)
	}
	defer rows.Close()

	var out []domain.Module
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *ModuleRepository) GetByCode(ctx context.Context, code string) (*domain.Module, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+moduleCols+" FROM saas.modules WHERE code=$1", code)
	m, err := scanModule(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrModuleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener módulo: %w", err)
	}
	return m, nil
}
