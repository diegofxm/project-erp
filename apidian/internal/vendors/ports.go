package vendors

import "context"

// CatalogPort valida LiabilityCodes y resuelve TaxSchemeName — mismo contrato que
// customers.CatalogPort (se satisface con el mismo catalogs.PostgresRepository).
type CatalogPort interface {
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
}
