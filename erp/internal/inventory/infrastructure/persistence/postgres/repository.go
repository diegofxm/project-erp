package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/inventory/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetStock(ctx context.Context, companyID, productID uuid.UUID, warehouse string) (*domain.StockEntry, error) {
	var e domain.StockEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, company_id, product_id, warehouse, quantity, updated_at
		 FROM inventory.stock
		 WHERE company_id=$1 AND product_id=$2 AND warehouse=$3`,
		companyID, productID, warehouse,
	).Scan(&e.ID, &e.CompanyID, &e.ProductID, &e.Warehouse, &e.Quantity, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener stock: %w", err)
	}
	return &e, nil
}

func (r *Repository) ListStock(ctx context.Context, companyID uuid.UUID) ([]domain.StockEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, product_id, warehouse, quantity, updated_at
		 FROM inventory.stock
		 WHERE company_id=$1 ORDER BY warehouse, product_id`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar stock: %w", err)
	}
	defer rows.Close()

	var out []domain.StockEntry
	for rows.Next() {
		var e domain.StockEntry
		if err := rows.Scan(&e.ID, &e.CompanyID, &e.ProductID, &e.Warehouse, &e.Quantity, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer stock: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertStock(ctx context.Context, e domain.StockEntry) error {
	e.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory.stock (id, company_id, product_id, warehouse, quantity, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (company_id, product_id, warehouse)
		 DO UPDATE SET quantity=$5, updated_at=$6`,
		e.ID, e.CompanyID, e.ProductID, e.Warehouse, e.Quantity, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("actualizar stock: %w", err)
	}
	return nil
}

func (r *Repository) SaveMovement(ctx context.Context, m domain.Movement) (*domain.Movement, error) {
	m.CreatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory.movements
		 (id, company_id, product_id, warehouse, type, quantity, reference, description, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.ID, m.CompanyID, m.ProductID, m.Warehouse, string(m.Type),
		m.Quantity, m.Reference, m.Description, m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar movimiento: %w", err)
	}
	return &m, nil
}

func (r *Repository) ListMovements(ctx context.Context, companyID uuid.UUID, productID *uuid.UUID) ([]domain.Movement, error) {
	var rows pgx.Rows
	var err error
	if productID != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, company_id, product_id, warehouse, type, quantity, reference, description, created_at
			 FROM inventory.movements
			 WHERE company_id=$1 AND product_id=$2 ORDER BY created_at DESC`,
			companyID, *productID,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, company_id, product_id, warehouse, type, quantity, reference, description, created_at
			 FROM inventory.movements
			 WHERE company_id=$1 ORDER BY created_at DESC`,
			companyID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listar movimientos: %w", err)
	}
	defer rows.Close()

	var out []domain.Movement
	for rows.Next() {
		var m domain.Movement
		var mType string
		if err := rows.Scan(
			&m.ID, &m.CompanyID, &m.ProductID, &m.Warehouse,
			&mType, &m.Quantity, &m.Reference, &m.Description, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("leer movimiento: %w", err)
		}
		m.Type = domain.MovementType(mType)
		out = append(out, m)
	}
	return out, rows.Err()
}
