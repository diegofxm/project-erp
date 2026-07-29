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
			entity_type_code, merchant_registration_number,
			name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
			department_code, municipality_code, address_line,
			address_city_name, address_state_name, address_country_code, address_country_name,
			email, phone,
			payment_terms_days, is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
		s.ID, s.CompanyID,
		s.IdentificationTypeCode, s.IdentificationNumber, s.CheckDigit,
		s.EntityTypeCode, s.MerchantRegistrationNumber,
		s.Name, s.TaxSchemeCode, s.TaxSchemeName, s.TaxRegimeCode, s.LiabilityCodes,
		s.DepartmentCode, s.MunicipalityCode, s.AddressLine,
		s.AddressCityName, s.AddressStateName, s.AddressCountryCode, s.AddressCountryName,
		s.Email, s.Phone,
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
			entity_type_code=$4, merchant_registration_number=$5,
			name=$6, tax_scheme_code=$7, tax_scheme_name=$8, tax_regime_code=$9, liability_codes=$10,
			department_code=$11, municipality_code=$12, address_line=$13,
			address_city_name=$14, address_state_name=$15, address_country_code=$16, address_country_name=$17,
			email=$18, phone=$19, payment_terms_days=$20, updated_at=NOW()
		WHERE id=$21 AND company_id=$22`,
		s.IdentificationTypeCode, s.IdentificationNumber, s.CheckDigit,
		s.EntityTypeCode, s.MerchantRegistrationNumber,
		s.Name, s.TaxSchemeCode, s.TaxSchemeName, s.TaxRegimeCode, s.LiabilityCodes,
		s.DepartmentCode, s.MunicipalityCode, s.AddressLine,
		s.AddressCityName, s.AddressStateName, s.AddressCountryCode, s.AddressCountryName,
		s.Email, s.Phone, s.PaymentTermsDays,
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
	       entity_type_code, merchant_registration_number,
	       name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
	       department_code, municipality_code, address_line,
	       address_city_name, address_state_name, address_country_code, address_country_name,
	       email, phone,
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
		&sup.EntityTypeCode, &sup.MerchantRegistrationNumber,
		&sup.Name, &sup.TaxSchemeCode, &sup.TaxSchemeName, &sup.TaxRegimeCode, &sup.LiabilityCodes,
		&sup.DepartmentCode, &sup.MunicipalityCode, &sup.AddressLine,
		&sup.AddressCityName, &sup.AddressStateName, &sup.AddressCountryCode, &sup.AddressCountryName,
		&sup.Email, &sup.Phone,
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
