package vendors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/apidian/internal/sqlutil"
	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx. Mismas columnas aplanadas que
// customers.PostgresRepository — tabla "vendors" en vez de "customers".
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de proveedores sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const vendorColumns = `
	id,
	issuer_id,
	entity_type_code,
	identification_number,
	identification_type_code,
	identification_verification_code,
	name,
	address_line,
	address_city_code,
	address_city_name,
	address_state_code,
	address_state_name,
	address_country_code,
	address_country_name,
	tax_scheme_code,
	tax_scheme_name,
	liability_codes,
	tax_regime_code,
	phone,
	email,
	merchant_registration_number,
	created_at,
	updated_at`

// Repository define las operaciones de persistencia del catálogo de proveedores.
type Repository interface {
	Create(ctx context.Context, v Vendor) (*Vendor, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Vendor, error)
	ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Vendor, error)
	Update(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Vendor, error)
	Delete(ctx context.Context, issuerID, id uuid.UUID) error
}

func (r *PostgresRepository) Create(ctx context.Context, v Vendor) (*Vendor, error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now

	args := partyArgs(v.ID, v.IssuerID, v.Party, v.CreatedAt, v.UpdatedAt)
	_, err := r.pool.Exec(ctx, `INSERT INTO vendors (`+vendorColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("create vendor: %w", err)
	}
	return &v, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Vendor, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+vendorColumns+` FROM vendors WHERE id = $1`, id)
	return scanVendor(row)
}

func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Vendor, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+vendorColumns+` FROM vendors WHERE issuer_id = $1 ORDER BY created_at DESC`, issuerID)
	if err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}
	defer rows.Close()

	var out []*Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Vendor, error) {
	liabilityCodes := party.LiabilityCodes
	if liabilityCodes == nil {
		liabilityCodes = []string{}
	}
	now := time.Now().UTC()

	tag, err := r.pool.Exec(ctx, `
		UPDATE vendors SET
			entity_type_code = $1,
			identification_number = $2,
			identification_type_code = $3,
			identification_verification_code = $4,
			name = $5,
			address_line = $6,
			address_city_code = $7,
			address_city_name = $8,
			address_state_code = $9,
			address_state_name = $10,
			address_country_code = $11,
			address_country_name = $12,
			tax_scheme_code = $13,
			tax_scheme_name = $14,
			liability_codes = $15,
			tax_regime_code = $16,
			phone = $17,
			email = $18,
			merchant_registration_number = $19,
			updated_at = $20
		WHERE id = $21 AND issuer_id = $22`,
		party.EntityTypeCode, party.Identification.Number, party.Identification.TypeCode, party.Identification.VerificationCode,
		party.Name, nullableString(party.Address.Line), nullableString(party.Address.CityCode), nullableString(party.Address.CityName),
		nullableString(party.Address.StateCode), nullableString(party.Address.StateName), nullableString(party.Address.CountryCode), nullableString(party.Address.CountryName),
		nullableString(party.TaxSchemeCode), nullableString(party.TaxSchemeName), liabilityCodes, nullableString(party.TaxRegimeCode), party.Phone, party.Email, party.MerchantRegistrationNumber,
		now, id, issuerID,
	)
	if err != nil {
		return nil, fmt.Errorf("update vendor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrVendorNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresRepository) Delete(ctx context.Context, issuerID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM vendors WHERE id = $1 AND issuer_id = $2`, id, issuerID)
	if err != nil {
		return fmt.Errorf("delete vendor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrVendorNotFound
	}
	return nil
}

func partyArgs(id, issuerID uuid.UUID, p domain.Party, createdAt, updatedAt time.Time) []any {
	liabilityCodes := p.LiabilityCodes
	if liabilityCodes == nil {
		liabilityCodes = []string{}
	}
	return []any{
		id, issuerID, p.EntityTypeCode, p.Identification.Number, p.Identification.TypeCode, p.Identification.VerificationCode,
		p.Name, nullableString(p.Address.Line), nullableString(p.Address.CityCode), nullableString(p.Address.CityName),
		nullableString(p.Address.StateCode), nullableString(p.Address.StateName), nullableString(p.Address.CountryCode), nullableString(p.Address.CountryName),
		nullableString(p.TaxSchemeCode), nullableString(p.TaxSchemeName), liabilityCodes, nullableString(p.TaxRegimeCode), p.Phone, p.Email, p.MerchantRegistrationNumber, createdAt, updatedAt,
	}
}

func scanVendor(row pgx.Row) (*Vendor, error) {
	var v Vendor
	var addrLine, addrCityCode, addrCityName, addrStateCode, addrStateName, addrCountryCode, addrCountryName *string
	var taxSchemeCode, taxSchemeName, taxRegimeCode *string

	err := row.Scan(
		&v.ID, &v.IssuerID, &v.Party.EntityTypeCode, &v.Party.Identification.Number, &v.Party.Identification.TypeCode, &v.Party.Identification.VerificationCode,
		&v.Party.Name, &addrLine, &addrCityCode, &addrCityName, &addrStateCode, &addrStateName, &addrCountryCode, &addrCountryName,
		&taxSchemeCode, &taxSchemeName, &v.Party.LiabilityCodes, &taxRegimeCode, &v.Party.Phone, &v.Party.Email, &v.Party.MerchantRegistrationNumber,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVendorNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan vendor: %w", err)
	}

	v.Party.TaxSchemeCode = deref(taxSchemeCode)
	v.Party.TaxSchemeName = deref(taxSchemeName)
	v.Party.TaxRegimeCode = deref(taxRegimeCode)
	v.Party.Address = domain.Address{
		Line:        deref(addrLine),
		CityCode:    deref(addrCityCode),
		CityName:    deref(addrCityName),
		StateCode:   deref(addrStateCode),
		StateName:   deref(addrStateName),
		CountryCode: deref(addrCountryCode),
		CountryName: deref(addrCountryName),
	}
	return &v, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
