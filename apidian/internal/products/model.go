package products

import (
	"time"

	"github.com/google/uuid"
)

// Product es un ítem/servicio guardado para reutilizar al armar líneas de un documento —
// catálogo de conveniencia, mismo espíritu que customers.Customer.
//
// Deliberadamente NO incluye Quantity, LineExtensionCents, Taxes (plural) ni ReferencePrice
// de domain.Line — esos son datos de USO (cuántas unidades, en qué factura, con qué impuestos
// aplicaron esa vez), no del catálogo. TaxTypeCode/TaxTypeName/TaxPercent son un default
// único de conveniencia, no una lista: si una factura real necesita más de un impuesto por
// línea, eso se decide al construir el lineDTO de la emisión, no aquí.
type Product struct {
	ID       uuid.UUID
	IssuerID uuid.UUID

	Description      string
	UnitCode         string // catálogo unit_measures
	UnitPriceCents   int64
	ItemCode         string
	ItemTypeCode     string // catálogo de estándares de codificación de producto
	ItemTypeName     string
	ItemTypeAgencyID string

	// Impuesto por defecto a aplicar cuando se use este producto en una línea — opcional.
	TaxTypeCode string // catálogo tax_types
	TaxTypeName string
	TaxPercent  float64

	CreatedAt time.Time
	UpdatedAt time.Time
}
