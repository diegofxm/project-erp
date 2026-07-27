package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type ConfirmUseCase struct{ repo domain.Repository }

func NewConfirmUseCase(repo domain.Repository) *ConfirmUseCase { return &ConfirmUseCase{repo: repo} }

func (uc *ConfirmUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	o, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.StatusDraft {
		return nil, domain.ErrPurchaseNotDraft
	}
	if err := uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusConfirmed); err != nil {
		return nil, err
	}
	o.Status = domain.StatusConfirmed
	return o, nil
}
