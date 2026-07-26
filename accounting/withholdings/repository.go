package withholdings

import "context"

// Repository abstrae el acceso a datos del catálogo de retenciones.
type Repository interface {
	// GetConcept devuelve el concepto más específico disponible para la combinación
	// (code, type, vendorType). Prioriza filas con applicable_to exacto sobre BOTH.
	GetConcept(ctx context.Context, code string, wType WithholdingType, vendorType VendorType) (*Concept, error)

	// ListConcepts devuelve todos los conceptos activos de un tipo dado.
	ListConcepts(ctx context.Context, wType WithholdingType) ([]*Concept, error)

	// GetUVT devuelve el valor del UVT para el año indicado.
	GetUVT(ctx context.Context, year int) (*UVTValue, error)

	// ListUVT devuelve todos los valores UVT registrados, ordenados por año desc.
	ListUVT(ctx context.Context) ([]*UVTValue, error)
}
