package withholdings

import (
	"time"

	"github.com/google/uuid"
)

// WithholdingType clasifica la retención según el impuesto que la origina.
type WithholdingType string

const (
	TypeRetefuente WithholdingType = "RETEFUENTE"
	TypeReteiva    WithholdingType = "RETEIVA"
	TypeReteica    WithholdingType = "RETEICA"
)

// VendorType distingue si el proveedor es persona natural o jurídica,
// porque la mayoría de conceptos tienen tarifas distintas según este criterio.
type VendorType string

const (
	VendorNatural  VendorType = "NATURAL"
	VendorJuridica VendorType = "JURIDICA"
)

// Concept es un concepto de retención del catálogo (tabla withholding_concepts).
// rate_bp se guarda en basis points: 250 = 2.5%.
type Concept struct {
	ID                uuid.UUID
	Code              string
	Name              string
	Type              WithholdingType
	RateBP            int     // basis points (1 bp = 0.01%)
	MinBaseUVT        float64 // UVT mínimos para que aplique la retención
	AccountPayable    string  // cuenta PUC cuando la empresa practica la retención
	AccountReceivable string  // cuenta PUC cuando le practican retención a la empresa
	ApplicableTo      string  // NATURAL, JURIDICA o BOTH
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Result es el resultado de calcular una retención sobre un pago o abono en cuenta.
type Result struct {
	ConceptCode       string
	ConceptName       string
	Type              WithholdingType
	RateBP            int
	Base              int64  // base gravable en centavos
	Amount            int64  // monto a retener en centavos
	AccountPayable    string // acreditar esta cuenta (empresa retiene)
	AccountReceivable string // debitar esta cuenta (le retienen a la empresa)
}

// UVTValue guarda el valor del UVT para un año fiscal específico.
type UVTValue struct {
	Year       int
	ValueCents int64
}
