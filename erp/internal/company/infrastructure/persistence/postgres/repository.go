package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/cryptutil"
)

type Repository struct {
	pool *pgxpool.Pool
	key  []byte // 32 bytes para AES-256
}

func NewRepository(pool *pgxpool.Pool, encryptionKey []byte) *Repository {
	return &Repository{pool: pool, key: encryptionKey}
}

func (r *Repository) Save(ctx context.Context, c domain.Company) (*domain.Company, error) {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	pinEnc, err := cryptutil.Encrypt(r.key, []byte(c.SoftwarePIN))
	if err != nil {
		return nil, fmt.Errorf("cifrar software_pin: %w", err)
	}
	certPwdEnc, err := cryptutil.Encrypt(r.key, []byte(c.CertificatePassword))
	if err != nil {
		return nil, fmt.Errorf("cifrar certificate_password: %w", err)
	}
	nePinEnc, err := cryptutil.Encrypt(r.key, []byte(c.NeSoftwarePIN))
	if err != nil {
		return nil, fmt.Errorf("cifrar ne_software_pin: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO company.companies (
			id, nit, check_digit, business_name, trade_name,
			identification_type_code, department_code, municipality_code, address_line, email, phone,
			environment, entity_type_code, tax_scheme_code, tax_scheme_name,
			liability_codes, tax_regime_code, industry_classification_codes, merchant_registration_number,
			software_id, software_pin_enc, certificate, certificate_password_enc,
			ne_software_id, ne_software_pin_enc,
			logo, logo_content_type, is_active, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,$11,
			$12,$13,$14,$15,
			$16,$17,$18,$19,
			$20,$21,$22,$23,
			$24,$25,
			$26,$27,$28,$29,$30
		)`,
		c.ID, c.NIT, c.CheckDigit, c.BusinessName, c.TradeName,
		c.IdentificationTypeCode, c.DepartmentCode, c.MunicipalityCode, c.AddressLine, c.Email, c.Phone,
		string(c.Environment), c.EntityTypeCode, c.TaxSchemeCode, c.TaxSchemeName,
		nilToEmpty(c.LiabilityCodes), c.TaxRegimeCode, nilToEmpty(c.IndustryClassificationCodes), c.MerchantRegistrationNumber,
		c.SoftwareID, pinEnc, c.Certificate, certPwdEnc,
		c.NeSoftwareID, nePinEnc,
		c.Logo, c.LogoContentType, c.IsActive, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar empresa: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return r.scanOne(ctx, "WHERE id = $1", id)
}

func (r *Repository) GetByNIT(ctx context.Context, nit string) (*domain.Company, error) {
	return r.scanOne(ctx, "WHERE nit = $1", nit)
}

func (r *Repository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Company, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		companySelect+" WHERE id = ANY($1) ORDER BY business_name",
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("listar empresas: %w", err)
	}
	defer rows.Close()

	var out []domain.Company
	for rows.Next() {
		c, err := r.scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateProfile(ctx context.Context, c domain.Company) (*domain.Company, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE company.companies SET
			business_name=$1, trade_name=$2,
			identification_type_code=$3, department_code=$4, municipality_code=$5,
			address_line=$6, email=$7, phone=$8,
			environment=$9, entity_type_code=$10, tax_scheme_code=$11, tax_scheme_name=$12,
			liability_codes=$13, tax_regime_code=$14,
			industry_classification_codes=$15, merchant_registration_number=$16,
			updated_at=NOW()
		WHERE id=$17`,
		c.BusinessName, c.TradeName,
		c.IdentificationTypeCode, c.DepartmentCode, c.MunicipalityCode,
		c.AddressLine, c.Email, c.Phone,
		string(c.Environment), c.EntityTypeCode, c.TaxSchemeCode, c.TaxSchemeName,
		c.LiabilityCodes, c.TaxRegimeCode,
		c.IndustryClassificationCodes, c.MerchantRegistrationNumber,
		c.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar perfil: %w", err)
	}
	return r.GetByID(ctx, c.ID)
}

func (r *Repository) UpdateCredentials(ctx context.Context, id uuid.UUID, softwareID, softwarePIN string, cert []byte, certPwd, neSoftwareID, neSoftwarePIN string) error {
	pinEnc, err := cryptutil.Encrypt(r.key, []byte(softwarePIN))
	if err != nil {
		return fmt.Errorf("cifrar software_pin: %w", err)
	}
	certPwdEnc, err := cryptutil.Encrypt(r.key, []byte(certPwd))
	if err != nil {
		return fmt.Errorf("cifrar certificate_password: %w", err)
	}
	nePinEnc, err := cryptutil.Encrypt(r.key, []byte(neSoftwarePIN))
	if err != nil {
		return fmt.Errorf("cifrar ne_software_pin: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE company.companies SET
			software_id=$1, software_pin_enc=$2, certificate=$3, certificate_password_enc=$4,
			ne_software_id=$5, ne_software_pin_enc=$6, updated_at=NOW()
		WHERE id=$7`,
		softwareID, pinEnc, cert, certPwdEnc, neSoftwareID, nePinEnc, id,
	)
	if err != nil {
		return fmt.Errorf("actualizar credenciales: %w", err)
	}
	return nil
}

func (r *Repository) UpdateLogo(ctx context.Context, id uuid.UUID, logo []byte, contentType string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE company.companies SET logo=$1, logo_content_type=$2, updated_at=NOW() WHERE id=$3",
		logo, contentType, id,
	)
	if err != nil {
		return fmt.Errorf("actualizar logo: %w", err)
	}
	return nil
}

func (r *Repository) DeleteLogo(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE company.companies SET logo=NULL, logo_content_type='', updated_at=NOW() WHERE id=$1",
		id,
	)
	if err != nil {
		return fmt.Errorf("eliminar logo: %w", err)
	}
	return nil
}

// --- helpers ---

const companySelect = `
	SELECT id, nit, check_digit, business_name, trade_name,
	       identification_type_code, department_code, municipality_code, address_line, email, phone,
	       environment, entity_type_code, tax_scheme_code, tax_scheme_name,
	       liability_codes, tax_regime_code, industry_classification_codes, merchant_registration_number,
	       software_id, software_pin_enc, certificate, certificate_password_enc,
	       ne_software_id, ne_software_pin_enc,
	       logo, logo_content_type, is_active, created_at, updated_at
	FROM company.companies`

func (r *Repository) scanOne(ctx context.Context, where string, arg any) (*domain.Company, error) {
	row := r.pool.QueryRow(ctx, companySelect+" "+where, arg)
	c, err := r.scanCompany(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCompanyNotFound
	}
	return c, err
}

type scanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanCompany(s scanner) (*domain.Company, error) {
	var c domain.Company
	var env string
	var pinEnc, certPwdEnc, nePinEnc []byte

	err := s.Scan(
		&c.ID, &c.NIT, &c.CheckDigit, &c.BusinessName, &c.TradeName,
		&c.IdentificationTypeCode, &c.DepartmentCode, &c.MunicipalityCode, &c.AddressLine, &c.Email, &c.Phone,
		&env, &c.EntityTypeCode, &c.TaxSchemeCode, &c.TaxSchemeName,
		&c.LiabilityCodes, &c.TaxRegimeCode, &c.IndustryClassificationCodes, &c.MerchantRegistrationNumber,
		&c.SoftwareID, &pinEnc, &c.Certificate, &certPwdEnc,
		&c.NeSoftwareID, &nePinEnc,
		&c.Logo, &c.LogoContentType, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("leer empresa: %w", err)
	}

	c.Environment = domain.Environment(env)

	pin, err := cryptutil.Decrypt(r.key, pinEnc)
	if err != nil {
		return nil, fmt.Errorf("descifrar software_pin: %w", err)
	}
	c.SoftwarePIN = string(pin)

	certPwd, err := cryptutil.Decrypt(r.key, certPwdEnc)
	if err != nil {
		return nil, fmt.Errorf("descifrar certificate_password: %w", err)
	}
	c.CertificatePassword = string(certPwd)

	nePin, err := cryptutil.Decrypt(r.key, nePinEnc)
	if err != nil {
		return nil, fmt.Errorf("descifrar ne_software_pin: %w", err)
	}
	c.NeSoftwarePIN = string(nePin)

	// Asegurar slices no nulos
	if c.LiabilityCodes == nil {
		c.LiabilityCodes = []string{}
	}
	if c.IndustryClassificationCodes == nil {
		c.IndustryClassificationCodes = []string{}
	}
	return &c, nil
}

func nilToEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
