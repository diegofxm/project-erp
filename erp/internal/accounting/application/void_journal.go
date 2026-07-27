package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type VoidJournalUseCase struct {
	journals domain.JournalRepository
}

func NewVoidJournalUseCase(journals domain.JournalRepository) *VoidJournalUseCase {
	return &VoidJournalUseCase{journals: journals}
}

func (uc *VoidJournalUseCase) Execute(ctx context.Context, id uuid.UUID) error {
	entry, err := uc.journals.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if errors.Is(err, domain.ErrJournalNotFound) {
		return domain.ErrJournalNotFound
	}
	if entry.Status == domain.StatusVoid {
		return domain.ErrJournalVoided
	}
	return uc.journals.Void(ctx, id)
}
