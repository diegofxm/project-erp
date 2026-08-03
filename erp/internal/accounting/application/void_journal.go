package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type VoidJournalUseCase struct {
	journals domain.JournalRepository
	periods  domain.PeriodRepository
}

func NewVoidJournalUseCase(journals domain.JournalRepository, periods domain.PeriodRepository) *VoidJournalUseCase {
	return &VoidJournalUseCase{journals: journals, periods: periods}
}

func (uc *VoidJournalUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) error {
	entry, err := uc.journals.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if entry.Status == domain.StatusVoid {
		return domain.ErrJournalVoided
	}
	period, err := uc.periods.GetByID(ctx, companyID, entry.PeriodID)
	if err != nil {
		return err
	}
	if period.Status == domain.PeriodClosed {
		return domain.ErrPeriodClosed
	}
	return uc.journals.Void(ctx, companyID, id)
}
