package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/supplier/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, s domain.Supplier) (*domain.Supplier, error) {
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO supplier.suppliers (
			id, company_id,
			identification_type_code, identification_number, check_digit,
			name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
			department_code, municipality_code, address_line, email, phone,
			payment_terms_days, is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		s.ID, s.CompanyID,
		s.IdentificationTypeCode, s.IdentificationNumber, s.CheckDigit,
		s.Name, s.TaxSchemeCode, s.TaxSchemeName, s.TaxRegimeCode, s.LiabilityCodes,
		s.DepartmentCode, s.MunicipalityCode, s.AddressLine, s.Email, s.Phone,
		s.PaymentTermsDays, s.IsActive, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar proveedor: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Supplier, error) {
	row := r.pool.QueryRow(ctx, supplierSelect+" WHERE id=$1 AND company_id=$2", id, companyID)
	s, err := scanSupplier(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSupplierNotFound
	}
	return s, err
}

func (r *Repository) GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*domain.Supplier, error) {
	row := r.pool.QueryRow(ctx,
		supplierSelect+" WHERE company_id=$1 AND identification_type_code=$2 AND identification_number=$3",
		companyID, identTypeCode, identNumber,
	)
	s, err := scanSupplier(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSupplierNotFound
	}
	return s, err
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Supplier, error) {
	rows, err := r.pool.Query(ctx,
		supplierSelect+" WHERE company_id=$1 ORDER BY name ASC",
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar proveedores: %w", err)
	}
	defer rows.Close()

	var out []domain.Supplier
	for rows.Next() {
		s, err := scanSupplier(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, s domain.Supplier) (*domain.Supplier, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE supplier.suppliers SET
			identification_type_code=$1, identification_number=$2, check_digit=$3,
			name=$4, tax_scheme_code=$5, tax_scheme_name=$6, tax_regime_code=$7, liability_codes=$8,
			department_code=$9, municipality_code=$10, address_line=$11, email=$12, phone=$13,
			payment_terms_days=$14, updated_at=NOW()
		WHERE id=$15 AND company_id=$16`,
		s.IdentificationTypeCode, s.IdentificationNumber, s.CheckDigit,
		s.Name, s.TaxSchemeCode, s.TaxSchemeName, s.TaxRegimeCode, s.LiabilityCodes,
		s.DepartmentCode, s.MunicipalityCode, s.AddressLine, s.Email, s.Phone,
		s.PaymentTermsDays,
		s.ID, s.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar proveedor: %w", err)
	}
	return r.GetByID(ctx, s.CompanyID, s.ID)
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM supplier.suppliers WHERE id=$1 AND company_id=$2",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("eliminar proveedor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSupplierNotFound
	}
	return nil
}

const supplierSelect = `
	SELECT id, company_id,
	       identification_type_code, identification_number, check_digit,
	       name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
	       department_code, municipality_code, address_line, email, phone,
	       payment_terms_days, is_active, created_at, updated_at
	FROM supplier.suppliers`

type scanner interface {
	Scan(dest ...any) error
}

func scanSupplier(s scanner) (*domain.Supplier, error) {
	var sup domain.Supplier
	err := s.Scan(
		&sup.ID, &sup.CompanyID,
		&sup.IdentificationTypeCode, &sup.IdentificationNumber, &sup.CheckDigit,
		&sup.Name, &sup.TaxSchemeCode, &sup.TaxSchemeName, &sup.TaxRegimeCode, &sup.LiabilityCodes,
		&sup.DepartmentCode, &sup.MunicipalityCode, &sup.AddressLine, &sup.Email, &sup.Phone,
		&sup.PaymentTermsDays, &sup.IsActive, &sup.CreatedAt, &sup.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leer proveedor: %w", err)
	}
	if sup.LiabilityCodes == nil {
		sup.LiabilityCodes = []string{}
	}
	return &sup, nil
}
