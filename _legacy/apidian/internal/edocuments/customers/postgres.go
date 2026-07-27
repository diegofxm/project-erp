package customers

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

// PostgresRepository implementa Repository usando pgx. domain.Party se guarda aplanado en
// columnas propias (mismo criterio que issuers.Issuer con su Address) — no como JSONB, porque
// a diferencia de documents.customer (snapshot legal, de solo lectura) este registro SÍ se
// actualiza con UPDATE, y columnas propias permiten filtrar/indexar más adelante si hace falta.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de clientes sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const customerColumns = `
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

func (r *PostgresRepository) Create(ctx context.Context, c Customer) (*Customer, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	liabilityCodes := c.Party.LiabilityCodes
	if liabilityCodes == nil {
		liabilityCodes = []string{}
	}

	args := partyArgs(c.ID, c.IssuerID, c.Party, c.CreatedAt, c.UpdatedAt)
	_, err := r.pool.Exec(ctx, `INSERT INTO edocuments.customers (`+customerColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+customerColumns+` FROM edocuments.customers WHERE id = $1`, id)
	return scanCustomer(row)
}

func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID) ([]*Customer, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+customerColumns+` FROM edocuments.customers WHERE issuer_id = $1 ORDER BY created_at DESC`, issuerID)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var out []*Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, issuerID, id uuid.UUID, party domain.Party) (*Customer, error) {
	liabilityCodes := party.LiabilityCodes
	if liabilityCodes == nil {
		liabilityCodes = []string{}
	}
	now := time.Now().UTC()

	tag, err := r.pool.Exec(ctx, `
		UPDATE edocuments.customers SET
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
		return nil, fmt.Errorf("update customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCustomerNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresRepository) Delete(ctx context.Context, issuerID, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM edocuments.customers WHERE id = $1 AND issuer_id = $2`, id, issuerID)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCustomerNotFound
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
		// tax_scheme_code es FK a tax_types(code) — una cadena vacía no es un código válido,
		// tiene que llegar como NULL de verdad cuando no se especifica (omitido del payload,
		// se completa después con defaults al emitir, ver documents.applyCustomerDefaults).
		nullableString(p.TaxSchemeCode), nullableString(p.TaxSchemeName), liabilityCodes, nullableString(p.TaxRegimeCode), p.Phone, p.Email, p.MerchantRegistrationNumber, createdAt, updatedAt,
	}
}

func scanCustomer(row pgx.Row) (*Customer, error) {
	var c Customer
	var addrLine, addrCityCode, addrCityName, addrStateCode, addrStateName, addrCountryCode, addrCountryName *string
	var taxSchemeCode, taxSchemeName, taxRegimeCode *string

	err := row.Scan(
		&c.ID, &c.IssuerID, &c.Party.EntityTypeCode, &c.Party.Identification.Number, &c.Party.Identification.TypeCode, &c.Party.Identification.VerificationCode,
		&c.Party.Name, &addrLine, &addrCityCode, &addrCityName, &addrStateCode, &addrStateName, &addrCountryCode, &addrCountryName,
		&taxSchemeCode, &taxSchemeName, &c.Party.LiabilityCodes, &taxRegimeCode, &c.Party.Phone, &c.Party.Email, &c.Party.MerchantRegistrationNumber,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan customer: %w", err)
	}

	c.Party.TaxSchemeCode = deref(taxSchemeCode)
	c.Party.TaxSchemeName = deref(taxSchemeName)
	c.Party.TaxRegimeCode = deref(taxRegimeCode)
	c.Party.Address = domain.Address{
		Line:        deref(addrLine),
		CityCode:    deref(addrCityCode),
		CityName:    deref(addrCityName),
		StateCode:   deref(addrStateCode),
		StateName:   deref(addrStateName),
		CountryCode: deref(addrCountryCode),
		CountryName: deref(addrCountryName),
	}
	return &c, nil
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
