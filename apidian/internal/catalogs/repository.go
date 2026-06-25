package catalogs

import "context"

// Repository define las operaciones de lectura del dominio de catálogos — interfaz angosta
// para poder probar con MemoryRepository en tests, mismo patrón que el resto de dominios
// (issuers.Repository, numbering.Repository, etc.). También cumple documents.CatalogPort/
// issuers.CatalogPort/customers.CatalogPort estructuralmente, sin que esos paquetes importen
// este (evita ciclos: documents/issuers/customers ya importan media apidian).
type Repository interface {
	ListDepartments(ctx context.Context) ([]Entry, error)
	ListMunicipalities(ctx context.Context, departmentCode string) ([]Municipality, error)
	ListIdentificationTypes(ctx context.Context) ([]Entry, error)
	ListTaxTypes(ctx context.Context) ([]Entry, error)
	ListPaymentMethods(ctx context.Context) ([]Entry, error)
	ListPaymentTerms(ctx context.Context) ([]Entry, error)
	ListUnitMeasures(ctx context.Context) ([]Entry, error)
	ListTaxRegimes(ctx context.Context) ([]Entry, error)
	ListLiabilityCodes(ctx context.Context) ([]Entry, error)
	ListDianDocumentTypes(ctx context.Context) ([]Entry, error)
	ListCurrencies(ctx context.Context) ([]Currency, error)

	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	// IsValidLiabilityCode — issuers.liability_codes/customers.liability_codes son TEXT[],
	// Postgres no soporta un FK contra cada elemento de un array nativamente, así que la
	// validación de "este código existe en el catálogo" se hace aquí, en código.
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
}
