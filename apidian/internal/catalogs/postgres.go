package catalogs

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository es el único tipo de este paquete — sin Service, ver model.go. Implementa
// documents.CatalogPort (IsValidPaymentTerm/IsValidPaymentMethod) estructuralmente, sin
// importar ese paquete (evita un ciclo: documents ya importa media apidian).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de catálogos sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// listEntries es el helper compartido por todos los catálogos con forma Entry (code/name/
// description). table SIEMPRE es un literal pasado por los métodos de este archivo, nunca
// algo derivado de un request — no hay riesgo de inyección por más que se interpole el
// nombre de tabla en la query.
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
	return r.listEntries(ctx, "departments")
}

func (r *PostgresRepository) ListIdentificationTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "identification_types")
}

func (r *PostgresRepository) ListTaxTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "tax_types")
}

func (r *PostgresRepository) ListPaymentMethods(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "payment_methods")
}

func (r *PostgresRepository) ListPaymentTerms(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "payment_terms")
}

func (r *PostgresRepository) ListUnitMeasures(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "unit_measures")
}

func (r *PostgresRepository) ListTaxRegimes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "tax_regimes")
}

func (r *PostgresRepository) ListLiabilityCodes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "liability_codes")
}

func (r *PostgresRepository) ListDianDocumentTypes(ctx context.Context) ([]Entry, error) {
	return r.listEntries(ctx, "dian_document_types")
}

func (r *PostgresRepository) ListCurrencies(ctx context.Context) ([]Currency, error) {
	rows, err := r.pool.Query(ctx, "SELECT code, name, symbol FROM currencies ORDER BY code")
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

// ListMunicipalities devuelve los municipios — si departmentCode no es "", filtra por
// departamento (lo que de verdad necesita un select dependiente departamento→municipio en un
// frontend; sin el filtro, sería forzar a transferir y filtrar 1.122 filas en el cliente).
func (r *PostgresRepository) ListMunicipalities(ctx context.Context, departmentCode string) ([]Municipality, error) {
	query := "SELECT code, name, department_code, description FROM municipalities"
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

// ListItemStandards devuelve las 4 filas fijas de la tabla 13.3.5 (sección 9.45) — usado por
// el frontend para construir el selector de estándar de clasificación de ítems.
func (r *PostgresRepository) ListItemStandards(ctx context.Context) ([]ItemStandard, error) {
	rows, err := r.pool.Query(ctx, "SELECT code, name, agency_id, description FROM item_standards ORDER BY code")
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

// IsValidPaymentTerm/IsValidPaymentMethod implementan documents.CatalogPort — ver el comentario
// en internal/documents/ports.go sobre por qué hace falta esto (payment_means vive en JSONB,
// nunca pudo tener un FK real).
func (r *PostgresRepository) IsValidPaymentTerm(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "payment_terms", code)
}

func (r *PostgresRepository) IsValidPaymentMethod(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "payment_methods", code)
}

// IsValidLiabilityCode implementa issuers.CatalogPort/customers.CatalogPort/
// documents.CatalogPort — ver el comentario en Repository sobre por qué hace falta esto
// (liability_codes es TEXT[], sin FK posible contra cada elemento).
func (r *PostgresRepository) IsValidLiabilityCode(ctx context.Context, code string) (bool, error) {
	return r.exists(ctx, "liability_codes", code)
}

// GetTaxTypeName implementa issuers.CatalogPort/customers.CatalogPort/products.CatalogPort/
// documents.CatalogPort — ver el comentario en Repository.
func (r *PostgresRepository) GetTaxTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "tax_types", code)
}

// GetPaymentTermName/GetPaymentMethodName implementan documents.CatalogPort — ver el
// comentario en Repository.
func (r *PostgresRepository) GetPaymentTermName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "payment_terms", code)
}

func (r *PostgresRepository) GetPaymentMethodName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "payment_methods", code)
}

func (r *PostgresRepository) GetIdentificationTypeName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "identification_types", code)
}

// GetItemStandardName implementa documents.CatalogPort/products.CatalogPort — ver el
// comentario en Repository (sección 9.45).
func (r *PostgresRepository) GetItemStandardName(ctx context.Context, code string) (string, bool, error) {
	return r.getName(ctx, "item_standards", code)
}

// GetItemStandardAgencyID — separado de GetItemStandardName porque agency_id puede ser ""
// con found=true (fila 999, donde el atributo no debe usarse en el XML).
func (r *PostgresRepository) GetItemStandardAgencyID(ctx context.Context, code string) (string, bool, error) {
	var agencyID string
	err := r.pool.QueryRow(ctx, "SELECT agency_id FROM item_standards WHERE code = $1", code).Scan(&agencyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("buscar agency_id en item_standards: %w", err)
	}
	return agencyID, true, nil
}

// getName es el helper compartido por GetTaxTypeName/GetPaymentTermName/GetPaymentMethodName
// — los tres son "busca el nombre de este código en este catálogo", solo cambia la tabla.
func (r *PostgresRepository) getName(ctx context.Context, table, code string) (string, bool, error) {
	var name string
	query := fmt.Sprintf("SELECT name FROM %s WHERE code = $1", table)
	err := r.pool.QueryRow(ctx, query, code).Scan(&name)
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
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE code = $1)", table)
	if err := r.pool.QueryRow(ctx, query, code).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar %s: %w", table, err)
	}
	return exists, nil
}
