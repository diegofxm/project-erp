package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type GetJournalUseCase struct {
	journals domain.JournalRepository
}

func NewGetJournalUseCase(journals domain.JournalRepository) *GetJournalUseCase {
	return &GetJournalUseCase{journals: journals}
}

func (uc *GetJournalUseCase) ByID(ctx context.Context, id uuid.UUID) (*domain.JournalEntry, error) {
	return uc.journals.GetByID(ctx, id)
}

func (uc *GetJournalUseCase) List(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*domain.JournalEntry, error) {
	return uc.journals.ListByCompany(ctx, companyID, limit, offset)
}

func (uc *GetJournalUseCase) BySourceDocument(ctx context.Context, companyID, sourceDocID uuid.UUID, sourceDocType string) ([]*domain.JournalEntry, error) {
	return uc.journals.GetBySourceDocument(ctx, companyID, sourceDocID, sourceDocType)
}

func (uc *GetJournalUseCase) PLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]domain.PLBalance, error) {
	return uc.journals.GetYearPLBalances(ctx, companyID, year)
}

func (uc *GetJournalUseCase) BSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]domain.PLBalance, error) {
	return uc.journals.GetBSBalances(ctx, companyID, asOf)
}
