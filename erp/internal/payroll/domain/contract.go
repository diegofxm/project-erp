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

type Contract struct {
	ID               uuid.UUID
	EmployeeID       uuid.UUID
	CompanyID        uuid.UUID
	ContractType     ContractType
	WorkSchedule     string
	Position         string
	CostCenter       string
	SalaryCents      int64
	SalaryType       SalaryType
	RiskClass        WorkRiskClass
	StartDate        time.Time
	EndDate          *time.Time
	TerminationDate  *time.Time
	TerminationCause string
	HealthEntity     string
	PensionEntity    string
	ARLEntity        string
	CajaEntity       string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateContractInput struct {
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
