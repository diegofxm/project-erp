package iva

import (
	"time"

	"github.com/google/uuid"
)

// TaxPeriodType define la frecuencia de declaración de IVA según la clasificación del contribuyente.
type TaxPeriodType string

const (
	PeriodBimestral     TaxPeriodType = "BIMESTRAL"     // cada 2 meses — grandes contribuyentes
	PeriodCuatrimestral TaxPeriodType = "CUATRIMESTRAL" // cada 4 meses — contribuyentes medianos
	PeriodAnual         TaxPeriodType = "ANUAL"          // anual — pequeños contribuyentes
)

// DeclarationStatus representa el estado del ciclo de vida de una declaración.
type DeclarationStatus string

const (
	StatusDraft     DeclarationStatus = "DRAFT"
	StatusFiled     DeclarationStatus = "FILED"
	StatusPaid      DeclarationStatus = "PAID"
	StatusCorrected DeclarationStatus = "CORRECTED"
)

// IVALine es el movimiento de IVA en una cuenta contable para el período del F300.
// Generated = créditos en la cuenta (IVA cobrado a clientes).
// Deductible = débitos en la cuenta (IVA pagado a proveedores).
type IVALine struct {
	AccountCode string
	AccountName string
	Generated   int64
	Deductible  int64
}

// ReteIVALine es el IVA retenido en cuentas 1365xx (reteiva que el cliente nos practicó).
// Reduce el IVA neto a pagar.
type ReteIVALine struct {
	AccountCode string
	AccountName string
	Withheld    int64
}

// Form300 es la vista preliminar del Formulario 300 antes de radicar.
// Los campos de "base gravable" deben obtenerse del IncomeStatement del mismo período.
type Form300 struct {
	CompanyID    uuid.UUID
	PeriodStart  time.Time
	PeriodEnd    time.Time
	PeriodType   TaxPeriodType

	// Detalle por cuenta contable
	IVALines    []*IVALine
	ReteIVALines []*ReteIVALine

	// Totales del formulario (en centavos)
	TotalGenerated  int64 // suma de IVALines.Generated
	TotalDeductible int64 // suma de IVALines.Deductible
	TotalWithheld   int64 // suma de ReteIVALines.Withheld (reteiva a favor)
	NetIVA          int64 // TotalGenerated - TotalDeductible - TotalWithheld
	PreviousBalance int64 // saldo a favor del período anterior (entrada manual)
	ToPay           int64 // max(0, NetIVA - PreviousBalance)
	CarryForward    int64 // max(0, -(NetIVA - PreviousBalance)) — saldo a favor siguiente
}

// IVADeclaration es el registro de una declaración de IVA presentada o en borrador.
type IVADeclaration struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	PeriodStart     time.Time
	PeriodEnd       time.Time
	PeriodType      TaxPeriodType
	GeneratedIVA    int64
	DeductibleIVA   int64
	WithheldIVA     int64
	NetIVA          int64
	PreviousBalance int64
	ToPay           int64
	CarryForward    int64
	Status          DeclarationStatus
	JournalID       uuid.UUID // uuid.Nil hasta que se pague
	FiledAt         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
