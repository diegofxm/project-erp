package catalogs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository sobre pgx. Sin Service — no hay lógica de
// negocio que orquestar, solo lecturas SQL. Los sub-paquetes de edocuments los reciben
// vía sus propias interfaces CatalogPort, nunca importan este tipo directamente.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// listEntries es el helper compartido por todos los catálogos con forma Entry.
// table es SIEMPRE un literal del código — nunca derivado de un request, sin riesgo de inyección.
func (r *PostgresRepository) listEntries(ctx context.Context, table string) ([]Entry, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf("SELECT code, name, description FROM %s ORDER BY code", table))
	if err != nil {
		return nil, fmt.Errorf("listar %s: %w", table, err)
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Code, &e.Name, &e.Description); err != nil {
			return nil, fmt.Errorf("leer %s: %w", table, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListDepartments(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.departments")
}

func (r *PostgresRepository) ListIdentificationTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.identification_types")
}

func (r *PostgresRepository) ListTaxTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.dian_tax_types")
}

func (r *PostgresRepository) ListPaymentMethods(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.payment_methods")
}

func (r *PostgresRepository) ListPaymentTerms(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.payment_terms")
}

func (r *PostgresRepository) ListUnitMeasures(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.unit_measures")
}

func (r *PostgresRepository) ListTaxRegimes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.tax_regimes")
}

func (r *PostgresRepository) ListLiabilityCodes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.liability_codes")
}

func (r *PostgresRepository) ListDianDocumentTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "catalogs.dian_document_types")
}

func (r *PostgresRepository) ListMunicipalities(ctx context.Context, departmentCode string) ([]Municipality, error) {
	query := "SELECT code, name, department_code, description FROM catalogs.municipalities"
	args := []any{}
	if departmentCode != "" {
		query += " WHERE department_code = $1"
		args = append(args, departmentCode)
	}
	query += " ORDER BY code"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar municipalities: %w", err)
	}
	defer rows.Close()
	var out []Municipality
	for rows.Next() {
		var m Municipality
		if err := rows.Scan(&m.Code, &m.Name, &m.DepartmentCode, &m.Description); err != nil {
			return nil, fmt.Errorf("leer municipalities: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListCurrencies(ctx context.Context) ([]Currency, error) {
	rows, err := r.pool.Query(ctx, "SELECT code, name, symbol FROM catalogs.currencies ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("listar currencies: %w", err)
	}
	defer rows.Close()
	var out []Currency
	for rows.Next() {
		var c Currency
		if err := rows.Scan(&c.Code, &c.Name, &c.Symbol); err != nil {
			return nil, fmt.Errorf("leer currencies: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListItemStandards(ctx context.Context) ([]ItemStandard, error) {
	rows, err := r.pool.Query(ctx, "SELECT code, name, agency_id, description FROM catalogs.item_standards ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("listar item_standards: %w", err)
	}
	defer rows.Close()
	var out []ItemStandard
	for rows.Next() {
		var s ItemStandard
		if err := rows.Scan(&s.Code, &s.Name, &s.AgencyID, &s.Description); err != nil {
			return nil, fmt.Errorf("leer item_standards: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListCiiuCodes(ctx context.Context) ([]CiiuCode, error) {
	rows, err := r.pool.Query(ctx, "SELECT code, description FROM catalogs.ciiu_codes ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("listar ciiu_codes: %w", err)
	}
	defer rows.Close()
	var out []CiiuCode
	for rows.Next() {
		var c CiiuCode
		if err := rows.Scan(&c.Code, &c.Description); err != nil {
			return nil, fmt.Errorf("leer ciiu_codes: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) IsValidPaymentTerm(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.payment_terms", code)
}

func (r *PostgresRepository) IsValidPaymentMethod(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.payment_methods", code)
}

func (r *PostgresRepository) IsValidLiabilityCode(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.liability_codes", code)
}

func (r *PostgresRepository) GetTaxTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.dian_tax_types", code)
}

func (r *PostgresRepository) GetPaymentTermName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.payment_terms", code)
}

func (r *PostgresRepository) GetPaymentMethodName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.payment_methods", code)
}

func (r *PostgresRepository) GetIdentificationTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.identification_types", code)
}

func (r *PostgresRepository) GetTaxRegimeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.tax_regimes", code)
}

func (r *PostgresRepository) GetLiabilityCodeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.liability_codes", code)
}

func (r *PostgresRepository) GetItemStandardName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.item_standards", code)
}

func (r *PostgresRepository) GetItemStandardAgencyID(ctx context.Context, code string) (string, bool, error) {
	var agencyID string
	err := r.pool.QueryRow(ctx, "SELECT agency_id FROM catalogs.item_standards WHERE code = $1", code).Scan(&agencyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar agency_id en item_standards: %w", err)
	}
	return agencyID, true, nil
}

func (r *PostgresRepository) GetCiiuDescription(ctx context.Context, code string) (string, bool, error) {
	var desc string
	err := r.pool.QueryRow(ctx, "SELECT description FROM catalogs.ciiu_codes WHERE code = $1", code).Scan(&desc)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar descripción en ciiu_codes: %w", err)
	}
	return desc, true, nil
}

func (r *PostgresRepository) getName(ctx context.Context, table, code string) (string, bool, error) {
	var name string
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT name FROM %s WHERE code = $1", table), code).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar nombre en %s: %w", table, err)
	}
	return name, true, nil
}

func (r *PostgresRepository) exists(ctx context.Context, table, code string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE code = $1)", table), code).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar %s: %w", table, err)
	}
	return exists, nil
}
