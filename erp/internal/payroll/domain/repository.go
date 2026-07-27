package domain

import (
	"context"

	"github.com/google/uuid"
)

type EmployeeRepository interface {
	Create(ctx context.Context, in CreateEmployeeInput) (*Employee, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Employee, error)
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*Employee, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}

type ContractRepository interface {
	Create(ctx context.Context, in CreateContractInput) (*Contract, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Contract, error)
	GetActive(ctx context.Context, employeeID uuid.UUID) (*Contract, error)
	ListByEmployee(ctx context.Context, employeeID uuid.UUID) ([]*Contract, error)
	Terminate(ctx context.Context, id uuid.UUID, cause string) error
}

type PayslipRepository interface {
	Create(ctx context.Context, in CreatePayslipInput, result CalcResult) (*Payslip, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Payslip, error)
	ListByCompany(ctx context.Context, companyID uuid.UUID, year, month int) ([]*Payslip, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status PayslipStatus) error
	GetSMMLV(ctx context.Context, year int) (int64, error)
	GetARLRate(ctx context.Context, year int, riskClass string) (int, error)
}
