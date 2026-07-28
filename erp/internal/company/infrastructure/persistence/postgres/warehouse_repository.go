package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/company/domain"
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type WarehouseRepository struct{ pool *pgxpool.Pool }

func NewWarehouseRepository(pool *pgxpool.Pool) *WarehouseRepository {
	return &WarehouseRepository{pool: pool}
}

var _ domain.WarehouseRepository = (*WarehouseRepository)(nil)

func (r *WarehouseRepository) Save(ctx context.Context, w domain.Warehouse) (*domain.Warehouse, error) {
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now
	_, err := r.pool.Exec(ctx, `
		INSERT INTO company.warehouses (id, company_id, code, name, address, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		w.ID, w.CompanyID, w.Code, w.Name, w.Address, w.IsActive, w.CreatedAt, w.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrWarehouseCodeTaken
		}
		return nil, fmt.Errorf("guardar bodega: %w", err)
	}
	return &w, nil
}

func (r *WarehouseRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error) {
	var w domain.Warehouse
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, code, name, address, is_active, created_at, updated_at
		FROM company.warehouses WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&w.ID, &w.CompanyID, &w.Code, &w.Name, &w.Address, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWarehouseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener bodega: %w", err)
	}
	return &w, nil
}

func (r *WarehouseRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Warehouse, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, code, name, address, is_active, created_at, updated_at
		FROM company.warehouses WHERE company_id=$1 ORDER BY code`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar bodegas: %w", err)
	}
	defer rows.Close()

	var out []domain.Warehouse
	for rows.Next() {
		var w domain.Warehouse
		if err := rows.Scan(&w.ID, &w.CompanyID, &w.Code, &w.Name, &w.Address, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer bodega: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *WarehouseRepository) Update(ctx context.Context, w domain.Warehouse) (*domain.Warehouse, error) {
	w.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE company.warehouses SET code=$1, name=$2, address=$3, updated_at=$4
		WHERE id=$5 AND company_id=$6`,
		w.Code, w.Name, w.Address, w.UpdatedAt, w.ID, w.CompanyID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrWarehouseCodeTaken
		}
		return nil, fmt.Errorf("actualizar bodega: %w", err)
	}
	return &w, nil
}

func (r *WarehouseRepository) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE company.warehouses SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND company_id=$2",
		id, companyID,
	)
	return err
}
