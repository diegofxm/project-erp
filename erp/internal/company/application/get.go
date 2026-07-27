package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

type GetUseCase struct {
	repo domain.Repository
}

func NewGetUseCase(repo domain.Repository) *GetUseCase {
	return &GetUseCase{repo: repo}
}

func (uc *GetUseCase) ByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *GetUseCase) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Company, error) {
	return uc.repo.ListByIDs(ctx, ids)
}
