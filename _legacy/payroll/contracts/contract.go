package contracts

import (
	"time"

	"github.com/google/uuid"
)

// ContractType identifica la modalidad de vinculación laboral colombiana.
type ContractType string

const (
	ContractTypeFijo       ContractType = "fijo"       // término fijo
	ContractTypeIndefinido ContractType = "indefinido"  // término indefinido
	ContractTypeObra       ContractType = "obra"        // obra o labor
	ContractTypeServicios  ContractType = "servicios"   // prestación de servicios
)

// SalaryType diferencia salario ordinario de salario integral.
// El salario integral (mínimo 10 SMMLV) ya incluye prestaciones sociales;
// no aplica auxilio de transporte ni los cálculos normales de aportes.
type SalaryType string

const (
	SalaryTypeOrdinary  SalaryType = "ordinary"
	SalaryTypeIntegral  SalaryType = "integral"
)

// WorkRiskClass es la clase de riesgo ARL asignada al cargo.
type WorkRiskClass string

const (
	RiskClassI   WorkRiskClass = "I"
	RiskClassII  WorkRiskClass = "II"
	RiskClassIII WorkRiskClass = "III"
	RiskClassIV  WorkRiskClass = "IV"
	RiskClassV   WorkRiskClass = "V"
)

// Contract representa el contrato laboral vigente de un empleado.
type Contract struct {
	ID                uuid.UUID
	EmployeeID        uuid.UUID
	CompanyID         uuid.UUID
	ContractType      ContractType
	WorkSchedule      string
	Position          string
	CostCenter        string
	SalaryCents       int64
	SalaryType        SalaryType
	RiskClass         WorkRiskClass
	StartDate         time.Time
	EndDate           *time.Time
	TerminationDate   *time.Time
	TerminationCause  string
	HealthEntity      string
	PensionEntity     string
	ARLEntity         string
	CajaEntity        string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// CreateInput contiene los datos para crear un contrato.
type CreateInput struct {
	EmployeeID    uuid.UUID
	CompanyID     uuid.UUID
	ContractType  ContractType
	WorkSchedule  string
	Position      string
	CostCenter    string
	SalaryCents   int64
	SalaryType    SalaryType
	RiskClass     WorkRiskClass
	StartDate     time.Time
	EndDate       *time.Time
	HealthEntity  string
	PensionEntity string
	ARLEntity     string
	CajaEntity    string
}
