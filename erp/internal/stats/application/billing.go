package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/stats/domain"
)

type GetBillingStatsUseCase struct {
	repo domain.Repository
}

func NewGetBillingStatsUseCase(repo domain.Repository) *GetBillingStatsUseCase {
	return &GetBillingStatsUseCase{repo: repo}
}

func (uc *GetBillingStatsUseCase) Execute(ctx context.Context, companyID uuid.UUID) (*domain.BillingStats, error) {
	return uc.repo.GetBillingStats(ctx, companyID)
}
