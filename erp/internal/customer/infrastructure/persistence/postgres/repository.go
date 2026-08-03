package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/customer/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO customer.customers (
			id, company_id,
			identification_type_code, identification_number, check_digit,
			entity_type_code, merchant_registration_number,
			name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
			department_code, municipality_code, address_line,
			address_city_name, address_state_name, address_country_code, address_country_name,
			email, phone,
			is_active, created_at, updated_at, credit_limit
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
		c.ID, c.CompanyID,
		c.IdentificationTypeCode, c.IdentificationNumber, c.CheckDigit,
		c.EntityTypeCode, c.MerchantRegistrationNumber,
		c.Name, c.TaxSchemeCode, c.TaxSchemeName, c.TaxRegimeCode, c.LiabilityCodes,
		c.DepartmentCode, c.MunicipalityCode, c.AddressLine,
		c.AddressCityName, c.AddressStateName, c.AddressCountryCode, c.AddressCountryName,
		c.Email, c.Phone,
		c.IsActive, c.CreatedAt, c.UpdatedAt, c.CreditLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar cliente: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Customer, error) {
	row := r.pool.QueryRow(ctx, customerSelect+" WHERE id=$1 AND company_id=$2", id, companyID)
	c, err := scanCustomer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCustomerNotFound
	}
	return c, err
}

func (r *Repository) GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*domain.Customer, error) {
	row := r.pool.QueryRow(ctx,
		customerSelect+" WHERE company_id=$1 AND identification_type_code=$2 AND identification_number=$3",
		companyID, identTypeCode, identNumber,
	)
	c, err := scanCustomer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCustomerNotFound
	}
	return c, err
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Customer, error) {
	rows, err := r.pool.Query(ctx,
		customerSelect+" WHERE company_id=$1 ORDER BY name ASC",
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar clientes: %w", err)
	}
	defer rows.Close()

	var out []domain.Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE customer.customers SET
			identification_type_code=$1, identification_number=$2, check_digit=$3,
			entity_type_code=$4, merchant_registration_number=$5,
			name=$6, tax_scheme_code=$7, tax_scheme_name=$8, tax_regime_code=$9, liability_codes=$10,
			department_code=$11, municipality_code=$12, address_line=$13,
			address_city_name=$14, address_state_name=$15, address_country_code=$16, address_country_name=$17,
			email=$18, phone=$19, updated_at=NOW(), credit_limit=$20
		WHERE id=$21 AND company_id=$22`,
		c.IdentificationTypeCode, c.IdentificationNumber, c.CheckDigit,
		c.EntityTypeCode, c.MerchantRegistrationNumber,
		c.Name, c.TaxSchemeCode, c.TaxSchemeName, c.TaxRegimeCode, c.LiabilityCodes,
		c.DepartmentCode, c.MunicipalityCode, c.AddressLine,
		c.AddressCityName, c.AddressStateName, c.AddressCountryCode, c.AddressCountryName,
		c.Email, c.Phone, c.CreditLimit, c.ID, c.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar cliente: %w", err)
	}
	return r.GetByID(ctx, c.CompanyID, c.ID)
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM customer.customers WHERE id=$1 AND company_id=$2",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("eliminar cliente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrCustomerNotFound
	}
	return nil
}

// --- helpers ---

const customerSelect = `
	SELECT id, company_id,
	       identification_type_code, identification_number, check_digit,
	       entity_type_code, merchant_registration_number,
	       name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
	       department_code, municipality_code, address_line,
	       address_city_name, address_state_name, address_country_code, address_country_name,
	       email, phone,
	       is_active, created_at, updated_at, credit_limit
	FROM customer.customers`

type scanner interface {
	Scan(dest ...any) error
}

func scanCustomer(s scanner) (*domain.Customer, error) {
	var c domain.Customer
	err := s.Scan(
		&c.ID, &c.CompanyID,
		&c.IdentificationTypeCode, &c.IdentificationNumber, &c.CheckDigit,
		&c.EntityTypeCode, &c.MerchantRegistrationNumber,
		&c.Name, &c.TaxSchemeCode, &c.TaxSchemeName, &c.TaxRegimeCode, &c.LiabilityCodes,
		&c.DepartmentCode, &c.MunicipalityCode, &c.AddressLine,
		&c.AddressCityName, &c.AddressStateName, &c.AddressCountryCode, &c.AddressCountryName,
		&c.Email, &c.Phone,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.CreditLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("leer cliente: %w", err)
	}
	if c.LiabilityCodes == nil {
		c.LiabilityCodes = []string{}
	}
	return &c, nil
}
