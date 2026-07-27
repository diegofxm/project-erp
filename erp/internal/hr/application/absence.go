package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/hr/domain"
)

type AbsenceUseCase struct {
	repo domain.AbsenceRepository
}

func NewAbsenceUseCase(repo domain.AbsenceRepository) *AbsenceUseCase {
	return &AbsenceUseCase{repo: repo}
}

func (uc *AbsenceUseCase) Request(ctx context.Context, in domain.CreateAbsenceInput) (*domain.Absence, error) {
	if !in.EndDate.After(in.StartDate) && !in.EndDate.Equal(in.StartDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return uc.repo.Create(ctx, in)
}

func (uc *AbsenceUseCase) Get(ctx context.Context, companyID, id uuid.UUID) (*domain.Absence, error) {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.CompanyID != companyID {
		return nil, domain.ErrAbsenceNotFound
	}
	return a, nil
}

func (uc *AbsenceUseCase) ListByEmployee(ctx context.Context, companyID, employeeID uuid.UUID) ([]*domain.Absence, error) {
	return uc.repo.ListByEmployee(ctx, companyID, employeeID)
}

func (uc *AbsenceUseCase) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*domain.Absence, error) {
	return uc.repo.ListByCompany(ctx, companyID)
}

func (uc *AbsenceUseCase) Approve(ctx context.Context, companyID, id uuid.UUID, notes string) error {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.CompanyID != companyID {
		return domain.ErrAbsenceNotFound
	}
	if a.Status != domain.AbsencePending {
		return domain.ErrAbsenceNotPending
	}
	return uc.repo.UpdateStatus(ctx, id, domain.AbsenceApproved, notes)
}

func (uc *AbsenceUseCase) Reject(ctx context.Context, companyID, id uuid.UUID, notes string) error {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a.CompanyID != companyID {
		return domain.ErrAbsenceNotFound
	}
	if a.Status != domain.AbsencePending {
		return domain.ErrAbsenceNotPending
	}
	return uc.repo.UpdateStatus(ctx, id, domain.AbsenceRejected, notes)
}
