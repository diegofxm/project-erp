package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type PostJournalRequest struct {
	CompanyID          uuid.UUID
	Date               time.Time
	Description        string
	Source             string
	EntryType          string
	VoucherType        string
	SourceDocumentID   uuid.UUID
	SourceDocumentType string
	Book               string
	Lines              []PostLineRequest
}

type PostLineRequest struct {
	AccountCode     string
	DebitCents      int64
	CreditCents     int64
	ThirdPartyNIT   string
	CostCenter      string
	Description     string
	ForeignAmount   int64
	ForeignCurrency string
}

type PostJournalUseCase struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
}

func NewPostJournalUseCase(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *PostJournalUseCase {
	return &PostJournalUseCase{accounts: accounts, periods: periods, journals: journals}
}

func (uc *PostJournalUseCase) Execute(ctx context.Context, req PostJournalRequest) (*domain.JournalEntry, error) {
	if len(req.Lines) < 2 {
		return nil, domain.ErrEmptyLines
	}

	lines := make([]*domain.JournalLine, len(req.Lines))
	var totalDebit, totalCredit int64

	for i, lr := range req.Lines {
		if (lr.DebitCents > 0) == (lr.CreditCents > 0) {
			return nil, fmt.Errorf("%w: línea %d (cuenta %s)", domain.ErrInvalidLine, i+1, lr.AccountCode)
		}
		acct, err := uc.accounts.GetPostable(ctx, lr.AccountCode)
		if err != nil {
			return nil, fmt.Errorf("línea %d, cuenta %q: %w", i+1, lr.AccountCode, err)
		}
		lines[i] = &domain.JournalLine{
			AccountID:       acct.ID,
			AccountCode:     acct.Code,
			Debit:           lr.DebitCents,
			Credit:          lr.CreditCents,
			ThirdPartyNIT:   lr.ThirdPartyNIT,
			CostCenter:      lr.CostCenter,
			Description:     lr.Description,
			ForeignAmount:   lr.ForeignAmount,
			ForeignCurrency: lr.ForeignCurrency,
		}
		totalDebit += lr.DebitCents
		totalCredit += lr.CreditCents
	}

	if totalDebit != totalCredit {
		return nil, fmt.Errorf("%w (débitos: %d, créditos: %d)", domain.ErrImbalancedEntry, totalDebit, totalCredit)
	}

	period, err := getOrCreatePeriod(ctx, uc.periods, req.CompanyID, req.Date)
	if err != nil {
		return nil, fmt.Errorf("obtener período: %w", err)
	}
	if period.Status == domain.PeriodClosed {
		return nil, domain.ErrPeriodClosed
	}

	entryType := domain.EntryType(req.EntryType)
	if entryType == "" {
		entryType = domain.EntryManual
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}
	book := domain.Book(req.Book)
	if book == "" {
		book = domain.BookBoth
	}

	entry := domain.JournalEntry{
		CompanyID:          req.CompanyID,
		PeriodID:           period.ID,
		Date:               req.Date,
		Description:        req.Description,
		Status:             domain.StatusPosted,
		Source:             source,
		EntryType:          entryType,
		VoucherType:        req.VoucherType,
		SourceDocumentID:   req.SourceDocumentID,
		SourceDocumentType: req.SourceDocumentType,
		Book:               book,
		Lines:              lines,
	}

	if req.VoucherType != "" {
		seq, err := uc.journals.NextVoucherSeq(ctx, req.CompanyID, req.VoucherType, req.Date.Year())
		if err != nil {
			return nil, fmt.Errorf("asignar comprobante: %w", err)
		}
		entry.VoucherNumber = fmt.Sprintf("%s-%d-%05d", req.VoucherType, req.Date.Year(), seq)
	}

	return uc.journals.Create(ctx, entry)
}

// getOrCreatePeriod devuelve el período para la fecha dada, creándolo si no existe.
func getOrCreatePeriod(ctx context.Context, repo domain.PeriodRepository, companyID uuid.UUID, date time.Time) (*domain.AccountingPeriod, error) {
	p, err := repo.GetByYearMonth(ctx, companyID, date.Year(), int(date.Month()))
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, domain.ErrPeriodNotFound) {
		return nil, err
	}
	return repo.Create(ctx, domain.AccountingPeriod{
		CompanyID: companyID,
		Year:      date.Year(),
		Month:     int(date.Month()),
	})
}
