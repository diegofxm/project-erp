package payslips

import (
	"time"

	"github.com/google/uuid"
)

// ConceptType clasifica cada línea de la liquidación.
type ConceptType string

const (
	// ConceptEarned: devengado — suma al bruto del empleado.
	ConceptEarned ConceptType = "earned"
	// ConceptDeduction: deducción — resta del bruto para obtener el neto.
	ConceptDeduction ConceptType = "deduction"
	// ConceptEmployerContribution: aporte parafiscal del empleador — no afecta el neto.
	ConceptEmployerContribution ConceptType = "employer_contribution"
)

// Status del ciclo de vida de la liquidación.
type Status string

const (
	StatusDraft    Status = "draft"
	StatusApproved Status = "approved"
	StatusPaid     Status = "paid"
	StatusVoided   Status = "voided"
)

// PayslipLine es una línea individual de la liquidación.
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

// Payslip es el encabezado de la liquidación de nómina de un empleado para un período.
type Payslip struct {
	ID                  uuid.UUID
	CompanyID           uuid.UUID
	EmployeeID          uuid.UUID
	ContractID          uuid.UUID
	PeriodYear          int
	PeriodMonth         int
	WorkedDays          int
	Status              Status
	TotalEarnedCents    int64
	TotalDeductedCents  int64
	NetPayCents         int64
	JournalID           *uuid.UUID
	PaidAt              *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Lines               []*PayslipLine
}

// CreateInput contiene los datos de entrada para generar una liquidación.
type CreateInput struct {
	CompanyID   uuid.UUID
	EmployeeID  uuid.UUID
	ContractID  uuid.UUID
	PeriodYear  int
	PeriodMonth int
	WorkedDays  int
}
