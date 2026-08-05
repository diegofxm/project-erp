package application

import (
	"context"
	"fmt"
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

func (uc *GetJournalUseCase) ByID(ctx context.Context, companyID, id uuid.UUID) (*domain.JournalEntry, error) {
	return uc.journals.GetByID(ctx, companyID, id)
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

func (uc *GetJournalUseCase) TrialBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]domain.TrialBalanceRow, error) {
	return uc.journals.GetTrialBalance(ctx, companyID, from, to)
}

func (uc *GetJournalUseCase) AccountLedger(ctx context.Context, companyID uuid.UUID, accountCode string, from, to time.Time) ([]domain.LedgerLine, error) {
	return uc.journals.GetAccountLedger(ctx, companyID, accountCode, from, to)
}

func (uc *GetJournalUseCase) RegisterVoucherType(ctx context.Context, cfg domain.VoucherTypeConfig) (*domain.VoucherTypeConfig, error) {
	return uc.journals.RegisterVoucherType(ctx, cfg)
}

func (uc *GetJournalUseCase) ListVoucherTypes(ctx context.Context, companyID uuid.UUID) ([]*domain.VoucherTypeConfig, error) {
	return uc.journals.ListVoucherTypes(ctx, companyID)
}

// SetVoucherCounter fija el próximo consecutivo a emitir para un tipo de comprobante — pensado
// para migrar una empresa que ya traía su propia numeración de otro sistema. Valida el código
// igual que PostJournalUseCase.Execute: los tipos estándar del sistema siempre son válidos, los
// personalizados deben estar registrados y activos.
func (uc *GetJournalUseCase) SetVoucherCounter(ctx context.Context, companyID uuid.UUID, code string, year, nextNumber int) (int, error) {
	if nextNumber < 1 {
		return 0, domain.ErrNumberCounterInvalid
	}
	if !domain.IsStandardVoucherType(code) {
		registered, err := uc.journals.IsRegisteredVoucherType(ctx, companyID, code)
		if err != nil {
			return 0, fmt.Errorf("validar tipo de comprobante: %w", err)
		}
		if !registered {
			return 0, fmt.Errorf("%w: %q", domain.ErrVoucherTypeUnknown, code)
		}
	}
	return uc.journals.SetVoucherCounter(ctx, companyID, code, year, nextNumber)
}
