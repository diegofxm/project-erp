package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
)

type CancelUseCase struct{ repo domain.Repository }

func NewCancelUseCase(repo domain.Repository) *CancelUseCase { return &CancelUseCase{repo: repo} }

func (uc *CancelUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) error {
	s, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if s.Status == domain.StatusCancelled {
		return domain.ErrSaleNotConfirmed
	}
	return uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusCancelled)
}
