// Package payroll gestiona el ciclo completo de nómina colombiana.
// Ver _legacy/payroll/ para la implementación de referencia:
//   - Calculator: función pura (sin DB) que aplica la ley laboral colombiana
//   - Conceptos: SALARIO, AUX_TRANSPORTE, SALUD_EMP, PENSION_EMP, SALUD_EMPL,
//     PENSION_EMPL, ARL (por clase de riesgo I-V), CAJA (4%), SENA (2%), ICBF (3%)
package domain

import (
	"time"

	"github.com/google/uuid"
)

type ContractType string

const (
	ContractFijo       ContractType = "fijo"
	ContractIndefinido ContractType = "indefinido"
	ContractObra       ContractType = "obra"
	ContractServicios  ContractType = "servicios"
)

type SalaryType string

const (
	SalaryOrdinary SalaryType = "ordinary"
	SalaryIntegral SalaryType = "integral"
)

type WorkRiskClass string

const (
	RiskClassI   WorkRiskClass = "I"
	RiskClassII  WorkRiskClass = "II"
	RiskClassIII WorkRiskClass = "III"
	RiskClassIV  WorkRiskClass = "IV"
	RiskClassV   WorkRiskClass = "V"
)

type Payslip struct {
	ID         uuid.UUID
	CompanyID  uuid.UUID
	EmployeeID uuid.UUID
	Period     string // "2026-01"

	WorkedDays  int
	SalaryCents int64

	TotalEarnedCents   int64
	TotalDeductedCents int64
	NetPayCents        int64

	Lines  []PayslipLine
	Status string // "draft", "approved", "paid", "voided"

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConceptType string

const (
	ConceptEarned              ConceptType = "earned"
	ConceptDeduction           ConceptType = "deduction"
	ConceptEmployerContribution ConceptType = "employer_contribution"
)

type PayslipLine struct {
	ConceptCode string
	ConceptName string
	ConceptType ConceptType
	AmountCents int64
}
