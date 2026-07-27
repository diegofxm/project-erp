package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/catalog/domain"
)

// Repository implementa domain.Repository sobre pgx.
// Sin capa de servicio — catálogos son datos de referencia sin lógica de negocio.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// listEntries es el helper compartido para catálogos con forma (code, name, description).
// table es siempre un literal de código — sin riesgo de inyección SQL.
func (r *Repository) listEntries(ctx context.Context, table string) ([]domain.Entry, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(
		"SELECT code, name, description FROM %s ORDER BY code", table,
	))
	if err != nil {
		return nil, fmt.Errorf("listar %s: %w", table, err)
	}
	defer rows.Close()
	var out []domain.Entry
	for rows.Next() {
		var e domain.Entry
		if err := rows.Scan(&e.Code, &e.Name, &e.Description); err != nil {
			return nil, fmt.Errorf("leer %s: %w", table, err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) ListDepartments(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.departments")
}

func (r *Repository) ListIdentificationTypes(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.identification_types")
}

func (r *Repository) ListTaxTypes(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.dian_tax_types")
}

func (r *Repository) ListPaymentMethods(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.payment_methods")
}

func (r *Repository) ListPaymentTerms(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.payment_terms")
}

func (r *Repository) ListUnitMeasures(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.unit_measures")
}

func (r *Repository) ListTaxRegimes(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.tax_regimes")
}

func (r *Repository) ListLiabilityCodes(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.liability_codes")
}

func (r *Repository) ListDianDocumentTypes(ctx context.Context) ([]domain.Entry, error) {
	return r.listEntries(ctx, "catalogs.dian_document_types")
}

func (r *Repository) ListMunicipalities(ctx context.Context, departmentCode string) ([]domain.Municipality, error) {
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
	var out []domain.Municipality
	for rows.Next() {
		var m domain.Municipality
		if err := rows.Scan(&m.Code, &m.Name, &m.DepartmentCode, &m.Description); err != nil {
			return nil, fmt.Errorf("leer municipalities: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) ListCurrencies(ctx context.Context) ([]domain.Currency, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT code, name, symbol FROM catalogs.currencies ORDER BY code",
	)
	if err != nil {
		return nil, fmt.Errorf("listar currencies: %w", err)
	}
	defer rows.Close()
	var out []domain.Currency
	for rows.Next() {
		var c domain.Currency
		if err := rows.Scan(&c.Code, &c.Name, &c.Symbol); err != nil {
			return nil, fmt.Errorf("leer currencies: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) ListItemStandards(ctx context.Context) ([]domain.ItemStandard, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT code, name, agency_id, description FROM catalogs.item_standards ORDER BY code",
	)
	if err != nil {
		return nil, fmt.Errorf("listar item_standards: %w", err)
	}
	defer rows.Close()
	var out []domain.ItemStandard
	for rows.Next() {
		var s domain.ItemStandard
		if err := rows.Scan(&s.Code, &s.Name, &s.AgencyID, &s.Description); err != nil {
			return nil, fmt.Errorf("leer item_standards: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) ListCiiuCodes(ctx context.Context) ([]domain.CiiuCode, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT code, description FROM catalogs.ciiu_codes ORDER BY code",
	)
	if err != nil {
		return nil, fmt.Errorf("listar ciiu_codes: %w", err)
	}
	defer rows.Close()
	var out []domain.CiiuCode
	for rows.Next() {
		var c domain.CiiuCode
		if err := rows.Scan(&c.Code, &c.Description); err != nil {
			return nil, fmt.Errorf("leer ciiu_codes: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// --- Validaciones puntuales ---

func (r *Repository) IsValidPaymentTerm(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.payment_terms", code)
}

func (r *Repository) IsValidPaymentMethod(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.payment_methods", code)
}

func (r *Repository) IsValidLiabilityCode(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "catalogs.liability_codes", code)
}

// --- Lookups por código ---

func (r *Repository) GetTaxTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.dian_tax_types", code)
}

func (r *Repository) GetPaymentTermName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.payment_terms", code)
}

func (r *Repository) GetPaymentMethodName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.payment_methods", code)
}

func (r *Repository) GetIdentificationTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.identification_types", code)
}

func (r *Repository) GetTaxRegimeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.tax_regimes", code)
}

func (r *Repository) GetLiabilityCodeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.liability_codes", code)
}

func (r *Repository) GetItemStandardName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "catalogs.item_standards", code)
}

func (r *Repository) GetItemStandardAgencyID(ctx context.Context, code string) (string, bool, error) {
	var agencyID string
	err := r.pool.QueryRow(ctx,
		"SELECT agency_id FROM catalogs.item_standards WHERE code = $1", code,
	).Scan(&agencyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar agency_id en item_standards: %w", err)
	}
	return agencyID, true, nil
}

func (r *Repository) GetCiiuDescription(ctx context.Context, code string) (string, bool, error) {
	var desc string
	err := r.pool.QueryRow(ctx,
		"SELECT description FROM catalogs.ciiu_codes WHERE code = $1", code,
	).Scan(&desc)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar descripción en ciiu_codes: %w", err)
	}
	return desc, true, nil
}

// --- helpers privados ---

func (r *Repository) getName(ctx context.Context, table, code string) (string, bool, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT name FROM %s WHERE code = $1", table), code,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar nombre en %s: %w", table, err)
	}
	return name, true, nil
}

func (r *Repository) exists(ctx context.Context, table, code string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE code = $1)", table), code,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("verificar %s: %w", table, err)
	}
	return ok, nil
}
