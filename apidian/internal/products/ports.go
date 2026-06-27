package products

import "context"

// CatalogPort resuelve TaxTypeName a partir de TaxTypeCode — el cliente ya no puede mandar el
// nombre, el servicio lo deriva del catálogo tax_types (ver docs/apidian-architecture.md,
// mismo patrón que issuers.CatalogPort/customers.CatalogPort).
type CatalogPort interface {
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
}
