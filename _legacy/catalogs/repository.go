package catalogs

import "context"

// Repository define las operaciones de lectura de los catálogos DIAN/DANE.
// Es read-only: ninguna operación modifica datos — los catálogos se cargan por Seed.
// Los sub-paquetes de edocuments (customers, products, vendors, documents, issuers) definen
// interfaces mínimas propias (CatalogPort) que este Repository satisface estructuralmente,
// sin que ellos importen este paquete — evita ciclos de importación.
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
	ListCiiuCodes(ctx context.Context) ([]CiiuCode, error)

	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)

	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
	GetPaymentTermName(ctx context.Context, code string) (name string, found bool, err error)
	GetPaymentMethodName(ctx context.Context, code string) (name string, found bool, err error)
	GetIdentificationTypeName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardAgencyID(ctx context.Context, code string) (agencyID string, found bool, err error)
	GetTaxRegimeName(ctx context.Context, code string) (name string, found bool, err error)
	GetLiabilityCodeName(ctx context.Context, code string) (name string, found bool, err error)
	GetCiiuDescription(ctx context.Context, code string) (description string, found bool, err error)
}
