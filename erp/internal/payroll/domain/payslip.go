// Package payroll gestiona el ciclo completo de nómina colombiana.
// Ver _legacy/payroll/ para la implementación de referencia.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConceptType string

const (
	ConceptEarned               ConceptType = "earned"
	ConceptDeduction            ConceptType = "deduction"
	ConceptEmployerContribution ConceptType = "employer_contribution"
)

type PayslipStatus string

const (
	PayslipDraft    PayslipStatus = "draft"
	PayslipApproved PayslipStatus = "approved"
	PayslipPaid     PayslipStatus = "paid"
	PayslipVoided   PayslipStatus = "voided"
)

type PayslipLine struct {
	ID          uuid.UUID
	PayslipID   uuid.UUID
	ConceptCode string
	ConceptName string
	ConceptType ConceptType
	Quantity    float64
	AmountCents int64
	CreatedAt   time.Time
}

type Payslip struct {
	ID                 uuid.UUID
	CompanyID          uuid.UUID
	Number             string // folio interno, ej. "NOM-2026-00001" — no exige numeración legal DIAN
	EmployeeID         uuid.UUID
	ContractID         uuid.UUID
	PeriodYear         int
	PeriodMonth        int
	WorkedDays         int
	Status             PayslipStatus
	TotalEarnedCents   int64
	TotalDeductedCents int64
	NetPayCents        int64
	JournalID          *uuid.UUID
	PaidAt             *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Lines              []*PayslipLine
}

type CreatePayslipInput struct {
	CompanyID   uuid.UUID
	EmployeeID  uuid.UUID
	ContractID  uuid.UUID
	PeriodYear  int
	PeriodMonth int
	WorkedDays  int
}

// PayrollGenerated se publica cuando un recibo de nómina pasa a estado "approved".
// accounting/ lo consume para registrar el gasto de personal.
type PayrollGenerated struct {
	PayslipID          uuid.UUID
	CompanyID          uuid.UUID
	EmployeeID         uuid.UUID
	PeriodYear         int
	PeriodMonth        int
	TotalEarnedCents   int64
	TotalDeductedCents int64
	NetPayCents        int64
}

func (PayrollGenerated) EventName() string { return "payroll.generated" }
