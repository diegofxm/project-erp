package journals

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/diegofxm/accounting/accounts"
	"github.com/diegofxm/accounting/periods"
	"github.com/google/uuid"
)

// Service encapsula el motor de partida doble.
type Service struct {
	repo        Repository
	accountsSvc *accounts.Service
	periodsSvc  *periods.Service
}

func NewService(repo Repository, accountsSvc *accounts.Service, periodsSvc *periods.Service) *Service {
	return &Service{
		repo:        repo,
		accountsSvc: accountsSvc,
		periodsSvc:  periodsSvc,
	}
}

// Post valida y registra un asiento contable. Reglas:
//  1. Mínimo 2 líneas.
//  2. Cada línea: exactamente uno de debit o credit > 0.
//  3. Cada código de cuenta existe, es activa y es posteable.
//  4. SUM(debit) == SUM(credit) (tolerancia de 1 centavo por redondeo).
//  5. Se obtiene (o crea) el periodo para la fecha del asiento.
func (s *Service) Post(ctx context.Context, req PostRequest) (*JournalEntry, error) {
	if len(req.Lines) < 2 {
		return nil, ErrEmptyLines
	}

	resolvedLines := make([]*JournalLine, len(req.Lines))
	var totalDebit, totalCredit float64

	for i, lr := range req.Lines {
		if (lr.Debit > 0) == (lr.Credit > 0) {
			return nil, fmt.Errorf("%w: línea %d (cuenta %s)", ErrInvalidLine, i+1, lr.AccountCode)
		}

		acct, err := s.accountsSvc.GetPostable(ctx, lr.AccountCode)
		if err != nil {
			return nil, fmt.Errorf("línea %d, cuenta %q: %w", i+1, lr.AccountCode, err)
		}

		resolvedLines[i] = &JournalLine{
			AccountID:   acct.ID,
			AccountCode: acct.Code,
			Debit:       lr.Debit,
			Credit:      lr.Credit,
			CostCenter:  lr.CostCenter,
			Description: lr.Description,
		}
		totalDebit += lr.Debit
		totalCredit += lr.Credit
	}

	if math.Abs(totalDebit-totalCredit) > 0.01 {
		return nil, fmt.Errorf("%w (débitos: %.2f, créditos: %.2f)", ErrImbalancedEntry, totalDebit, totalCredit)
	}

	period, err := s.periodsSvc.GetOrCreate(ctx, req.CompanyID, req.Date)
	if err != nil {
		return nil, fmt.Errorf("obtener periodo: %w", err)
	}
	if period.Status == periods.StatusClosed {
		return nil, periods.ErrPeriodClosed
	}

	entryType := req.EntryType
	if entryType == "" {
		entryType = EntryManual
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}

	return s.repo.Create(ctx, JournalEntry{
		CompanyID:   req.CompanyID,
		PeriodID:    period.ID,
		Date:        req.Date,
		Description: req.Description,
		Status:      StatusPosted,
		Source:      source,
		EntryType:   entryType,
		Lines:       resolvedLines,
	})
}

// Get devuelve un asiento por UUID incluyendo sus líneas.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*JournalEntry, error) {
	return s.repo.GetByID(ctx, id)
}

// Void anula un asiento POSTED. No elimina — el ledger es inmutable.
func (s *Service) Void(ctx context.Context, id uuid.UUID) error {
	entry, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if errors.Is(err, ErrJournalNotFound) {
		return ErrJournalNotFound
	}
	if entry.Status == StatusVoid {
		return ErrJournalVoided
	}
	return s.repo.Void(ctx, id)
}

// List devuelve los asientos de una empresa (cabecera, sin líneas) paginados.
func (s *Service) List(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListByCompany(ctx, companyID, limit, offset)
}
