package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type CancelUseCase struct{ repo domain.Repository }

func NewCancelUseCase(repo domain.Repository) *CancelUseCase { return &CancelUseCase{repo: repo} }

func (uc *CancelUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) error {
	o, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if o.Status == domain.StatusCancelled || o.Status == domain.StatusReceived {
		return domain.ErrPurchaseNotDraft
	}
	return uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusCancelled)
}
