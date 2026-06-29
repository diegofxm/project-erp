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
	ListItemStandards(ctx context.Context) ([]ItemStandard, error)

	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	// IsValidLiabilityCode — issuers.liability_codes/customers.liability_codes son TEXT[],
	// Postgres no soporta un FK contra cada elemento de un array nativamente, así que la
	// validación de "este código existe en el catálogo" se hace aquí, en código.
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)

	// GetTaxTypeName resuelve el nombre real de un código de tax_types — found=false (sin
	// error) si el código no existe, igual criterio que los IsValid*: el llamador decide qué
	// error de dominio lanzar. Existe para que issuers/customers/products/documents puedan
	// derivar *_name desde *_code en vez de confiar en que el cliente los mande coherentes
	// (ver docs/apidian-architecture.md).
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)

	// GetPaymentTermName/GetPaymentMethodName/GetIdentificationTypeName — mismo patrón que
	// GetTaxTypeName, para que la representación gráfica en PDF (internal/pdf, ver sección
	// 9.39) muestre "Contado"/"Transferencia.../"Cédula de Ciudadanía" en vez del código
	// numérico crudo, sin que documents tenga que adivinar el texto.
	GetPaymentTermName(ctx context.Context, code string) (name string, found bool, err error)
	GetPaymentMethodName(ctx context.Context, code string) (name string, found bool, err error)
	GetIdentificationTypeName(ctx context.Context, code string) (name string, found bool, err error)

	// GetItemStandardName/GetItemStandardAgencyID — ver ItemStandard (sección 9.45). Separados
	// porque AgencyID puede ser "" con found=true (fila 999), a diferencia del resto de
	// Get*Name donde found=false ya cubre "no hay nada que mostrar".
	GetItemStandardName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardAgencyID(ctx context.Context, code string) (agencyID string, found bool, err error)
}
