package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/application"
	"github.com/diegofxm/erp/internal/accounting/domain"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeAccountRepo struct {
	postable map[string]domain.Account
}

func (f *fakeAccountRepo) GetByCode(_ context.Context, code string) (*domain.Account, error) {
	return f.GetPostable(context.Background(), code)
}
func (f *fakeAccountRepo) GetPostable(_ context.Context, code string) (*domain.Account, error) {
	a, ok := f.postable[code]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return &a, nil
}
func (f *fakeAccountRepo) List(_ context.Context) ([]domain.Account, error) { return nil, nil }

type fakePeriodRepo struct {
	period domain.AccountingPeriod
}

func (f *fakePeriodRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.AccountingPeriod, error) {
	return &f.period, nil
}
func (f *fakePeriodRepo) GetByYearMonth(context.Context, uuid.UUID, int, int) (*domain.AccountingPeriod, error) {
	return &f.period, nil
}
func (f *fakePeriodRepo) Create(_ context.Context, p domain.AccountingPeriod) (*domain.AccountingPeriod, error) {
	return &f.period, nil
}
func (f *fakePeriodRepo) Close(context.Context, uuid.UUID, uuid.UUID) error     { return nil }
func (f *fakePeriodRepo) Reopen(context.Context, uuid.UUID, uuid.UUID) error    { return nil }
func (f *fakePeriodRepo) CloseAllForYear(context.Context, uuid.UUID, int) error { return nil }
func (f *fakePeriodRepo) List(context.Context, uuid.UUID) ([]domain.AccountingPeriod, error) {
	return nil, nil
}

type fakeJournalRepo struct {
	registeredVoucherTypes map[string]bool
	created                *domain.JournalEntry
}

func (f *fakeJournalRepo) Create(_ context.Context, e domain.JournalEntry) (*domain.JournalEntry, error) {
	e.ID = uuid.New()
	f.created = &e
	return &e, nil
}
func (f *fakeJournalRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.JournalEntry, error) {
	return nil, domain.ErrJournalNotFound
}
func (f *fakeJournalRepo) Void(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeJournalRepo) ListByCompany(context.Context, uuid.UUID, int, int) ([]*domain.JournalEntry, error) {
	return nil, nil
}
func (f *fakeJournalRepo) GetBySourceDocument(context.Context, uuid.UUID, uuid.UUID, string) ([]*domain.JournalEntry, error) {
	return nil, nil
}
func (f *fakeJournalRepo) NextVoucherSeq(context.Context, uuid.UUID, string, int) (int, error) {
	return 1, nil
}
func (f *fakeJournalRepo) SetVoucherCounter(context.Context, uuid.UUID, string, int, int) (int, error) {
	return 1, nil
}
func (f *fakeJournalRepo) GetYearPLBalances(context.Context, uuid.UUID, int) ([]domain.PLBalance, error) {
	return nil, nil
}
func (f *fakeJournalRepo) GetBSBalances(context.Context, uuid.UUID, time.Time) ([]domain.PLBalance, error) {
	return nil, nil
}
func (f *fakeJournalRepo) GetTrialBalance(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.TrialBalanceRow, error) {
	return nil, nil
}
func (f *fakeJournalRepo) GetIncomeInPeriod(context.Context, uuid.UUID, time.Time, time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeJournalRepo) GetAccountLedger(context.Context, uuid.UUID, string, time.Time, time.Time) ([]domain.LedgerLine, error) {
	return nil, nil
}
func (f *fakeJournalRepo) RegisterVoucherType(_ context.Context, cfg domain.VoucherTypeConfig) (*domain.VoucherTypeConfig, error) {
	return &cfg, nil
}
func (f *fakeJournalRepo) ListVoucherTypes(context.Context, uuid.UUID) ([]*domain.VoucherTypeConfig, error) {
	return nil, nil
}
func (f *fakeJournalRepo) IsRegisteredVoucherType(_ context.Context, _ uuid.UUID, code string) (bool, error) {
	return f.registeredVoucherTypes[code], nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newTestUseCase() (*application.PostJournalUseCase, *fakeAccountRepo, *fakePeriodRepo, *fakeJournalRepo) {
	accounts := &fakeAccountRepo{postable: map[string]domain.Account{
		"110505": {ID: uuid.New(), Code: "110505", Name: "Caja", Category: "Activo", IsPosting: true},
		"413595": {ID: uuid.New(), Code: "413595", Name: "Ingresos", Category: "Ingreso", IsPosting: true},
	}}
	periods := &fakePeriodRepo{period: domain.AccountingPeriod{ID: uuid.New(), Status: domain.PeriodOpen}}
	journals := &fakeJournalRepo{registeredVoucherTypes: map[string]bool{}}
	uc := application.NewPostJournalUseCase(accounts, periods, journals)
	return uc, accounts, periods, journals
}

func baseRequest() application.PostJournalRequest {
	return application.PostJournalRequest{
		CompanyID:   uuid.New(),
		Date:        time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		Description: "test",
		Lines: []application.PostLineRequest{
			{AccountCode: "110505", DebitCents: 1000},
			{AccountCode: "413595", CreditCents: 1000},
		},
	}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestPostJournal_RejectsImbalancedEntry(t *testing.T) {
	uc, _, _, _ := newTestUseCase()
	req := baseRequest()
	req.Lines[1].CreditCents = 999 // ya no cuadra contra 1000 del débito

	_, err := uc.Execute(context.Background(), req)
	if !errors.Is(err, domain.ErrImbalancedEntry) {
		t.Fatalf("esperaba ErrImbalancedEntry, got %v", err)
	}
}

func TestPostJournal_RejectsLineWithBothDebitAndCredit(t *testing.T) {
	uc, _, _, _ := newTestUseCase()
	req := baseRequest()
	req.Lines[0].CreditCents = 1000 // línea con débito Y crédito a la vez, inválida

	_, err := uc.Execute(context.Background(), req)
	if !errors.Is(err, domain.ErrInvalidLine) {
		t.Fatalf("esperaba ErrInvalidLine, got %v", err)
	}
}

func TestPostJournal_RejectsFewerThanTwoLines(t *testing.T) {
	uc, _, _, _ := newTestUseCase()
	req := baseRequest()
	req.Lines = req.Lines[:1]

	_, err := uc.Execute(context.Background(), req)
	if !errors.Is(err, domain.ErrEmptyLines) {
		t.Fatalf("esperaba ErrEmptyLines, got %v", err)
	}
}

func TestPostJournal_RejectsClosedPeriod(t *testing.T) {
	uc, _, periods, _ := newTestUseCase()
	periods.period.Status = domain.PeriodClosed

	_, err := uc.Execute(context.Background(), baseRequest())
	if !errors.Is(err, domain.ErrPeriodClosed) {
		t.Fatalf("esperaba ErrPeriodClosed, got %v", err)
	}
}

func TestPostJournal_RejectsNonPostableAccount(t *testing.T) {
	uc, _, _, _ := newTestUseCase()
	req := baseRequest()
	req.Lines[0].AccountCode = "999999" // no existe en el fake

	_, err := uc.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("esperaba error por cuenta inexistente, got nil")
	}
}

func TestPostJournal_AcceptsStandardVoucherTypeWithoutRegistration(t *testing.T) {
	uc, _, _, journals := newTestUseCase()
	req := baseRequest()
	req.VoucherType = domain.VoucherIncome // "CI" — tipo del sistema, no está en registeredVoucherTypes

	entry, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("no esperaba error para tipo estándar del sistema: %v", err)
	}
	if entry.VoucherNumber == "" {
		t.Fatal("esperaba que se generara un número de comprobante")
	}
	_ = journals
}

func TestPostJournal_RejectsUnregisteredCustomVoucherType(t *testing.T) {
	uc, _, _, _ := newTestUseCase()
	req := baseRequest()
	req.VoucherType = "ZZ99" // ni estándar ni registrado

	_, err := uc.Execute(context.Background(), req)
	if !errors.Is(err, domain.ErrVoucherTypeUnknown) {
		t.Fatalf("esperaba ErrVoucherTypeUnknown, got %v", err)
	}
}

func TestPostJournal_AcceptsRegisteredCustomVoucherType(t *testing.T) {
	uc, _, _, journals := newTestUseCase()
	journals.registeredVoucherTypes["FC"] = true
	req := baseRequest()
	req.VoucherType = "FC"

	entry, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("no esperaba error para tipo personalizado registrado: %v", err)
	}
	if entry.VoucherType != "FC" {
		t.Fatalf("esperaba voucher_type FC, got %q", entry.VoucherType)
	}
}
