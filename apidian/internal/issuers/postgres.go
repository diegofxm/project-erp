package issuers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/apidian/internal/cryptutil"
	"github.com/diegofxm/apidian/internal/sqlutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx. Cifra software_pin, certificate y
// certificate_password con AES-256-GCM (secrets.go) antes de persistir — nunca quedan en
// texto plano en la base de datos.
type PostgresRepository struct {
	pool *pgxpool.Pool
	key  []byte // 32 bytes, AES-256 — ver config.Config.IssuerSecretsKey
}

// NewPostgresRepository crea el repositorio de emisores sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool, secretsKey []byte) *PostgresRepository {
	return &PostgresRepository{pool: pool, key: secretsKey}
}

const issuerColumns = `
	id, nit, check_digit, business_name, trade_name, identification_type_code,
	department_code, municipality_code, address_line, email, phone, environment,
	entity_type_code, tax_scheme_code, tax_scheme_name, liability_codes, tax_regime_code,
	industry_classification_codes, merchant_registration_number,
	software_id, software_pin, certificate, certificate_password, is_active,
	created_at, updated_at`

// issuerSelectWithNames hace JOIN con departments/municipalities para traer
// DepartmentName/MunicipalityName — la DIAN exige el nombre, no solo el código, en
// cac:Address (ver el comentario en model.go). Solo se usa para lectura, nunca para INSERT.
const issuerSelectWithNames = `
	SELECT i.id, i.nit, i.check_digit, i.business_name, i.trade_name, i.identification_type_code,
	       i.department_code, i.municipality_code, d.name, m.name, i.address_line, i.email, i.phone, i.environment,
	       i.entity_type_code, i.tax_scheme_code, i.tax_scheme_name, i.liability_codes, i.tax_regime_code,
	       i.industry_classification_codes, i.merchant_registration_number,
	       i.software_id, i.software_pin, i.certificate, i.certificate_password, i.is_active,
	       i.created_at, i.updated_at
	FROM issuers i
	JOIN departments d ON d.code = i.department_code
	JOIN municipalities m ON m.code = i.municipality_code`

func (r *PostgresRepository) Create(ctx context.Context, iss Issuer) (*Issuer, error) {
	if iss.ID == uuid.Nil {
		iss.ID = uuid.New()
	}
	now := time.Now().UTC()
	iss.CreatedAt = now
	iss.UpdatedAt = now

	encPIN, err := cryptutil.Encrypt(r.key, []byte(iss.SoftwarePIN))
	if err != nil {
		return nil, fmt.Errorf("cifrar software_pin: %w", err)
	}
	encCert, err := cryptutil.Encrypt(r.key, iss.Certificate)
	if err != nil {
		return nil, fmt.Errorf("cifrar certificate: %w", err)
	}
	encCertPwd, err := cryptutil.Encrypt(r.key, []byte(iss.CertificatePassword))
	if err != nil {
		return nil, fmt.Errorf("cifrar certificate_password: %w", err)
	}

	// liability_codes es NOT NULL en el esquema (DEFAULT '{}') — pero ese default solo
	// aplica si la columna se omite del INSERT, no si se manda un nil explícito, así que un
	// slice nil de Go hay que convertirlo a vacío aquí, no confiar en el default de SQL.
	liabilityCodes := iss.LiabilityCodes
	if liabilityCodes == nil {
		liabilityCodes = []string{}
	}
	industryClassificationCodes := iss.IndustryClassificationCodes
	if industryClassificationCodes == nil {
		industryClassificationCodes = []string{}
	}

	args := []any{
		iss.ID, iss.NIT, iss.CheckDigit, iss.BusinessName, iss.TradeName, iss.IdentificationTypeCode,
		iss.DepartmentCode, iss.MunicipalityCode, iss.AddressLine, iss.Email, iss.Phone, string(iss.Environment),
		iss.EntityTypeCode, iss.TaxSchemeCode, iss.TaxSchemeName, liabilityCodes, iss.TaxRegimeCode,
		industryClassificationCodes, iss.MerchantRegistrationNumber,
		iss.SoftwareID, encPIN, encCert, encCertPwd, iss.IsActive, iss.CreatedAt, iss.UpdatedAt,
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO issuers (`+issuerColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrNITAlreadyExists
		}
		return nil, fmt.Errorf("create issuer: %w", err)
	}
	return &iss, nil
}

// Update persiste software_id/software_pin/certificate/certificate_password — los únicos
// campos editables vía UpdateIssuer (ver service.go), cifrando PIN/certificado/password con
// el mismo esquema que Create.
func (r *PostgresRepository) Update(ctx context.Context, iss Issuer) (*Issuer, error) {
	encPIN, err := cryptutil.Encrypt(r.key, []byte(iss.SoftwarePIN))
	if err != nil {
		return nil, fmt.Errorf("cifrar software_pin: %w", err)
	}
	encCert, err := cryptutil.Encrypt(r.key, iss.Certificate)
	if err != nil {
		return nil, fmt.Errorf("cifrar certificate: %w", err)
	}
	encCertPwd, err := cryptutil.Encrypt(r.key, []byte(iss.CertificatePassword))
	if err != nil {
		return nil, fmt.Errorf("cifrar certificate_password: %w", err)
	}

	// software_id se manda tal cual (nunca como NULL explícito) aunque la columna lo permita
	// — igual que Create: iss.SoftwareID == "" se guarda como cadena vacía. scan() lee la
	// columna directo a un string de Go (no *string); un NULL real ahí rompería el Scan.
	tag, err := r.pool.Exec(ctx, `
		UPDATE issuers SET
			software_id = $1, software_pin = $2, certificate = $3, certificate_password = $4, updated_at = $5
		WHERE id = $6`,
		iss.SoftwareID, encPIN, encCert, encCertPwd, time.Now().UTC(), iss.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update issuer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrIssuerNotFound
	}
	return r.GetByID(ctx, iss.ID)
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	row := r.pool.QueryRow(ctx, issuerSelectWithNames+` WHERE i.id = $1`, id)
	return r.scan(row)
}

func (r *PostgresRepository) GetByNIT(ctx context.Context, nit string) (*Issuer, error) {
	row := r.pool.QueryRow(ctx, issuerSelectWithNames+` WHERE i.nit = $1`, nit)
	return r.scan(row)
}

func (r *PostgresRepository) scan(row pgx.Row) (*Issuer, error) {
	var iss Issuer
	var env string
	var encPIN, encCert, encCertPwd []byte

	err := row.Scan(&iss.ID, &iss.NIT, &iss.CheckDigit, &iss.BusinessName, &iss.TradeName,
		&iss.IdentificationTypeCode, &iss.DepartmentCode, &iss.MunicipalityCode,
		&iss.DepartmentName, &iss.MunicipalityName, &iss.AddressLine,
		&iss.Email, &iss.Phone, &env, &iss.EntityTypeCode, &iss.TaxSchemeCode, &iss.TaxSchemeName,
		&iss.LiabilityCodes, &iss.TaxRegimeCode, &iss.IndustryClassificationCodes,
		&iss.MerchantRegistrationNumber, &iss.SoftwareID, &encPIN, &encCert, &encCertPwd,
		&iss.IsActive, &iss.CreatedAt, &iss.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrIssuerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan issuer: %w", err)
	}
	iss.Environment = Environment(env)

	pin, err := cryptutil.Decrypt(r.key, encPIN)
	if err != nil {
		return nil, fmt.Errorf("descifrar software_pin: %w", err)
	}
	iss.SoftwarePIN = string(pin)

	cert, err := cryptutil.Decrypt(r.key, encCert)
	if err != nil {
		return nil, fmt.Errorf("descifrar certificate: %w", err)
	}
	iss.Certificate = cert

	certPwd, err := cryptutil.Decrypt(r.key, encCertPwd)
	if err != nil {
		return nil, fmt.Errorf("descifrar certificate_password: %w", err)
	}
	iss.CertificatePassword = string(certPwd)

	return &iss, nil
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
