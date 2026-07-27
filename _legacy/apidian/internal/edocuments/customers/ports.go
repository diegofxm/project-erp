package customers

import "context"

// CatalogPort valida LiabilityCodes contra el catálogo liability_codes — TEXT[], sin FK
// posible contra cada elemento (mismo motivo y mismo patrón que issuers.CatalogPort/
// documents.CatalogPort). GetTaxTypeName resuelve TaxSchemeName a partir de TaxSchemeCode —
// el cliente ya no puede mandar el nombre, el servicio lo deriva del catálogo (ver
// docs/apidian-architecture.md).
type CatalogPort interface {
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
}
