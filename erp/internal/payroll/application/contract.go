package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/payroll/domain"
)

type ContractUseCase struct {
	repo domain.ContractRepository
	emps domain.EmployeeRepository
}

func NewContractUseCase(repo domain.ContractRepository, emps domain.EmployeeRepository) *ContractUseCase {
	return &ContractUseCase{repo: repo, emps: emps}
}

func (uc *ContractUseCase) Create(ctx context.Context, companyID uuid.UUID, in domain.CreateContractInput) (*domain.Contract, error) {
	emp, err := uc.emps.GetByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp.CompanyID != companyID {
		return nil, domain.ErrEmployeeNotFound
	}
	in.CompanyID = companyID
	return uc.repo.Create(ctx, in)
}

func (uc *ContractUseCase) Get(ctx context.Context, companyID, id uuid.UUID) (*domain.Contract, error) {
	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.CompanyID != companyID {
		return nil, domain.ErrContractNotFound
	}
	return c, nil
}

func (uc *ContractUseCase) ListByEmployee(ctx context.Context, companyID, employeeID uuid.UUID) ([]*domain.Contract, error) {
	emp, err := uc.emps.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp.CompanyID != companyID {
		return nil, domain.ErrEmployeeNotFound
	}
	return uc.repo.ListByEmployee(ctx, employeeID)
}

func (uc *ContractUseCase) Terminate(ctx context.Context, companyID, id uuid.UUID, cause string) error {
	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c.CompanyID != companyID {
		return domain.ErrContractNotFound
	}
	return uc.repo.Terminate(ctx, id, cause)
}
