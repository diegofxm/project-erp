package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/product/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, p domain.Product) (*domain.Product, error) {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO product.products (
			id, company_id, code, name, description,
			unit_measure_code, standard_code, standard_code_type,
			is_service, tax_scheme_code, tax_scheme_name, tax_rate,
			base_price, is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		p.ID, p.CompanyID, p.Code, p.Name, p.Description,
		p.UnitMeasureCode, p.StandardCode, p.StandardCodeType,
		p.IsService, p.TaxSchemeCode, p.TaxSchemeName, p.TaxRate,
		p.BasePrice, p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar producto: %w", err)
	}
	return &p, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Product, error) {
	row := r.pool.QueryRow(ctx, productSelect+" WHERE id=$1 AND company_id=$2", id, companyID)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	return p, err
}

func (r *Repository) GetByCode(ctx context.Context, companyID uuid.UUID, code string) (*domain.Product, error) {
	row := r.pool.QueryRow(ctx,
		productSelect+" WHERE company_id=$1 AND code=$2",
		companyID, code,
	)
	p, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrProductNotFound
	}
	return p, err
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Product, error) {
	rows, err := r.pool.Query(ctx,
		productSelect+" WHERE company_id=$1 ORDER BY name ASC",
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar productos: %w", err)
	}
	defer rows.Close()

	var out []domain.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, p domain.Product) (*domain.Product, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE product.products SET
			code=$1, name=$2, description=$3,
			unit_measure_code=$4, standard_code=$5, standard_code_type=$6,
			is_service=$7, tax_scheme_code=$8, tax_scheme_name=$9, tax_rate=$10,
			base_price=$11, updated_at=NOW()
		WHERE id=$12 AND company_id=$13`,
		p.Code, p.Name, p.Description,
		p.UnitMeasureCode, p.StandardCode, p.StandardCodeType,
		p.IsService, p.TaxSchemeCode, p.TaxSchemeName, p.TaxRate,
		p.BasePrice,
		p.ID, p.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar producto: %w", err)
	}
	return r.GetByID(ctx, p.CompanyID, p.ID)
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM product.products WHERE id=$1 AND company_id=$2",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("eliminar producto: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

const productSelect = `
	SELECT id, company_id, code, name, description,
	       unit_measure_code, standard_code, standard_code_type,
	       is_service, tax_scheme_code, tax_scheme_name, tax_rate,
	       base_price, is_active, created_at, updated_at
	FROM product.products`

type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(s scanner) (*domain.Product, error) {
	var p domain.Product
	err := s.Scan(
		&p.ID, &p.CompanyID, &p.Code, &p.Name, &p.Description,
		&p.UnitMeasureCode, &p.StandardCode, &p.StandardCodeType,
		&p.IsService, &p.TaxSchemeCode, &p.TaxSchemeName, &p.TaxRate,
		&p.BasePrice, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leer producto: %w", err)
	}
	return &p, nil
}
