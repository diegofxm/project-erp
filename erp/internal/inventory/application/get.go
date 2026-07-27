package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/inventory/domain"
)

type GetUseCase struct {
	repo domain.Repository
}

func NewGetUseCase(repo domain.Repository) *GetUseCase {
	return &GetUseCase{repo: repo}
}

func (uc *GetUseCase) Stock(ctx context.Context, companyID uuid.UUID) ([]domain.StockEntry, error) {
	return uc.repo.ListStock(ctx, companyID)
}

func (uc *GetUseCase) Movements(ctx context.Context, companyID uuid.UUID, productID *uuid.UUID) ([]domain.Movement, error) {
	return uc.repo.ListMovements(ctx, companyID, productID)
}
