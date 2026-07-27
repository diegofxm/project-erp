package products

import "context"

// CatalogPort resuelve TaxTypeName a partir de TaxTypeCode — el cliente ya no puede mandar el
// nombre, el servicio lo deriva del catálogo tax_types (ver docs/apidian-architecture.md,
// mismo patrón que issuers.CatalogPort/customers.CatalogPort).
//
// GetItemStandardName/GetItemStandardAgencyID resuelven ItemTypeName/ItemTypeAgencyID a partir
// de ItemTypeCode (tabla 13.3.5, sección 9.45) — AgencyID separado porque puede ser "" con
// found=true (fila "999", estándar propio del contribuyente).
type CatalogPort interface {
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardAgencyID(ctx context.Context, code string) (agencyID string, found bool, err error)
}
