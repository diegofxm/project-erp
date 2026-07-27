package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type GetUseCase struct{ repo domain.Repository }

func NewGetUseCase(repo domain.Repository) *GetUseCase { return &GetUseCase{repo: repo} }

func (uc *GetUseCase) ByID(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	return uc.repo.GetByID(ctx, companyID, id)
}

func (uc *GetUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.PurchaseOrder, error) {
	return uc.repo.List(ctx, companyID)
}
