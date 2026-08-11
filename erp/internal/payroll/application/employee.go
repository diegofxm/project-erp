package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/payroll/domain"
)

type EmployeeUseCase struct {
	repo domain.EmployeeRepository
}

func NewEmployeeUseCase(repo domain.EmployeeRepository) *EmployeeUseCase {
	return &EmployeeUseCase{repo: repo}
}

func (uc *EmployeeUseCase) Create(ctx context.Context, in domain.CreateEmployeeInput) (*domain.Employee, error) {
	return uc.repo.Create(ctx, in)
}

func (uc *EmployeeUseCase) Get(ctx context.Context, companyID, id uuid.UUID) (*domain.Employee, error) {
	e, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.CompanyID != companyID {
		return nil, domain.ErrEmployeeNotFound
	}
	return e, nil
}

func (uc *EmployeeUseCase) List(ctx context.Context, companyID uuid.UUID) ([]*domain.Employee, error) {
	return uc.repo.ListByCompany(ctx, companyID)
}

func (uc *EmployeeUseCase) Update(ctx context.Context, companyID, id uuid.UUID, in domain.UpdateEmployeeInput) (*domain.Employee, error) {
	e, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.CompanyID != companyID {
		return nil, domain.ErrEmployeeNotFound
	}
	return uc.repo.Update(ctx, id, in)
}

func (uc *EmployeeUseCase) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	e, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if e.CompanyID != companyID {
		return domain.ErrEmployeeNotFound
	}
	return uc.repo.Deactivate(ctx, id)
}
