package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type ManagePeriodUseCase struct {
	periods domain.PeriodRepository
}

func NewManagePeriodUseCase(periods domain.PeriodRepository) *ManagePeriodUseCase {
	return &ManagePeriodUseCase{periods: periods}
}

func (uc *ManagePeriodUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.AccountingPeriod, error) {
	return uc.periods.List(ctx, companyID)
}

func (uc *ManagePeriodUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.AccountingPeriod, error) {
	return uc.periods.GetByID(ctx, id)
}

func (uc *ManagePeriodUseCase) Close(ctx context.Context, id uuid.UUID) error {
	p, err := uc.periods.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p.Status == domain.PeriodClosed {
		return domain.ErrPeriodClosed
	}
	return uc.periods.Close(ctx, id)
}

func (uc *ManagePeriodUseCase) CloseYear(ctx context.Context, companyID uuid.UUID, year int) error {
	return uc.periods.CloseAllForYear(ctx, companyID, year)
}
