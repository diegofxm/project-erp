package application

import (
	"context"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type SettingsUseCase struct {
	repo domain.SettingsRepository
}

func NewSettingsUseCase(repo domain.SettingsRepository) *SettingsUseCase {
	return &SettingsUseCase{repo: repo}
}

func (uc *SettingsUseCase) Get(ctx context.Context) (*domain.Settings, error) {
	return uc.repo.Get(ctx)
}

func (uc *SettingsUseCase) Update(ctx context.Context, s domain.Settings) (*domain.Settings, error) {
	return uc.repo.Update(ctx, s)
}
