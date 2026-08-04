package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, p domain.Party) (*domain.Party, error) {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.pool.Exec(ctx, `
		INSERT INTO thirdparty.parties (
			id, company_id,
			identification_type_code, identification_number, check_digit,
			entity_type_code, merchant_registration_number,
			name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
			department_code, municipality_code, address_line,
			address_city_name, address_state_name, address_country_code, address_country_name,
			email, phone,
			is_customer, is_supplier, credit_limit, payment_terms_days,
			is_active, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)`,
		p.ID, p.CompanyID,
		p.IdentificationTypeCode, p.IdentificationNumber, p.CheckDigit,
		p.EntityTypeCode, p.MerchantRegistrationNumber,
		p.Name, p.TaxSchemeCode, p.TaxSchemeName, p.TaxRegimeCode, p.LiabilityCodes,
		p.DepartmentCode, p.MunicipalityCode, p.AddressLine,
		p.AddressCityName, p.AddressStateName, p.AddressCountryCode, p.AddressCountryName,
		p.Email, p.Phone,
		p.IsCustomer, p.IsSupplier, p.CreditLimit, p.PaymentTermsDays,
		p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar tercero: %w", err)
	}
	return &p, nil
}

func (r *Repository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Party, error) {
	row := r.pool.QueryRow(ctx, partySelect+" WHERE id=$1 AND company_id=$2", id, companyID)
	p, err := scanParty(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPartyNotFound
	}
	return p, err
}

func (r *Repository) GetByIdentification(ctx context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*domain.Party, error) {
	row := r.pool.QueryRow(ctx,
		partySelect+" WHERE company_id=$1 AND identification_type_code=$2 AND identification_number=$3",
		companyID, identTypeCode, identNumber,
	)
	p, err := scanParty(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrPartyNotFound
	}
	return p, err
}

func (r *Repository) List(ctx context.Context, companyID uuid.UUID, role domain.Role) ([]domain.Party, error) {
	q := partySelect + " WHERE company_id=$1"
	switch role {
	case domain.RoleCustomer:
		q += " AND is_customer"
	case domain.RoleSupplier:
		q += " AND is_supplier"
	}
	q += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("listar terceros: %w", err)
	}
	defer rows.Close()

	var out []domain.Party
	for rows.Next() {
		p, err := scanParty(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, p domain.Party) (*domain.Party, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE thirdparty.parties SET
			identification_type_code=$1, identification_number=$2, check_digit=$3,
			entity_type_code=$4, merchant_registration_number=$5,
			name=$6, tax_scheme_code=$7, tax_scheme_name=$8, tax_regime_code=$9, liability_codes=$10,
			department_code=$11, municipality_code=$12, address_line=$13,
			address_city_name=$14, address_state_name=$15, address_country_code=$16, address_country_name=$17,
			email=$18, phone=$19,
			is_customer=$20, is_supplier=$21, credit_limit=$22, payment_terms_days=$23,
			updated_at=NOW()
		WHERE id=$24 AND company_id=$25`,
		p.IdentificationTypeCode, p.IdentificationNumber, p.CheckDigit,
		p.EntityTypeCode, p.MerchantRegistrationNumber,
		p.Name, p.TaxSchemeCode, p.TaxSchemeName, p.TaxRegimeCode, p.LiabilityCodes,
		p.DepartmentCode, p.MunicipalityCode, p.AddressLine,
		p.AddressCityName, p.AddressStateName, p.AddressCountryCode, p.AddressCountryName,
		p.Email, p.Phone,
		p.IsCustomer, p.IsSupplier, p.CreditLimit, p.PaymentTermsDays,
		p.ID, p.CompanyID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar tercero: %w", err)
	}
	return r.GetByID(ctx, p.CompanyID, p.ID)
}

func (r *Repository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM thirdparty.parties WHERE id=$1 AND company_id=$2",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("eliminar tercero: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPartyNotFound
	}
	return nil
}

// --- helpers ---

const partySelect = `
	SELECT id, company_id,
	       identification_type_code, identification_number, check_digit,
	       entity_type_code, merchant_registration_number,
	       name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
	       department_code, municipality_code, address_line,
	       address_city_name, address_state_name, address_country_code, address_country_name,
	       email, phone,
	       is_customer, is_supplier, credit_limit, payment_terms_days,
	       is_active, created_at, updated_at
	FROM thirdparty.parties`

type scanner interface {
	Scan(dest ...any) error
}

func scanParty(s scanner) (*domain.Party, error) {
	var p domain.Party
	err := s.Scan(
		&p.ID, &p.CompanyID,
		&p.IdentificationTypeCode, &p.IdentificationNumber, &p.CheckDigit,
		&p.EntityTypeCode, &p.MerchantRegistrationNumber,
		&p.Name, &p.TaxSchemeCode, &p.TaxSchemeName, &p.TaxRegimeCode, &p.LiabilityCodes,
		&p.DepartmentCode, &p.MunicipalityCode, &p.AddressLine,
		&p.AddressCityName, &p.AddressStateName, &p.AddressCountryCode, &p.AddressCountryName,
		&p.Email, &p.Phone,
		&p.IsCustomer, &p.IsSupplier, &p.CreditLimit, &p.PaymentTermsDays,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leer tercero: %w", err)
	}
	if p.LiabilityCodes == nil {
		p.LiabilityCodes = []string{}
	}
	return &p, nil
}
