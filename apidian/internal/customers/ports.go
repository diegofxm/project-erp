package customers

import "context"

// CatalogPort valida LiabilityCodes contra el catálogo liability_codes — TEXT[], sin FK
// posible contra cada elemento (mismo motivo y mismo patrón que issuers.CatalogPort/
// documents.CatalogPort).
type CatalogPort interface {
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
}
