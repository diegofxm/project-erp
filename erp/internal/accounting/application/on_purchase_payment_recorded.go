package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/diegofxm/erp/internal/accounting/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPurchasePaymentRecorded registra el asiento contable cuando se paga a un proveedor.
//
// PUC colombiano:
//   - DB 2205   Proveedores nacionales (abono a cuentas por pagar)
//   - CR 110505 Caja general / 111005 Bancos (según medio de pago)
type OnPurchasePaymentRecorded struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
	log      *slog.Logger
}

func NewOnPurchasePaymentRecorded(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
	log *slog.Logger,
) *OnPurchasePaymentRecorded {
	return &OnPurchasePaymentRecorded{accounts: accounts, periods: periods, journals: journals, log: log}
}

func (h *OnPurchasePaymentRecorded) Register(bus *events.Bus) {
	bus.Subscribe(purchasedomain.PurchasePaymentRecorded{}.EventName(), func(ctx context.Context, evt events.Event) error {
		ev, ok := evt.(purchasedomain.PurchasePaymentRecorded)
		if !ok {
			return nil
		}
		if err := h.handle(ctx, ev); err != nil {
			h.log.Error("asiento pago compra", "payment_id", ev.PaymentID, "error", err)
			return fmt.Errorf("accounting: asiento pago compra %s: %w", ev.PaymentID, err)
		}
		return nil
	})
}

func (h *OnPurchasePaymentRecorded) handle(ctx context.Context, ev purchasedomain.PurchasePaymentRecorded) error {
	amountCents := toCents(ev.Amount)

	acctPayable, err := h.accounts.GetPostable(ctx, "2205")
	if err != nil {
		return err
	}
	acctCash, err := h.accounts.GetPostable(ctx, cashAccountCode(string(ev.PaymentMethod)))
	if err != nil {
		return err
	}

	lines := []*domain.JournalLine{
		{
			AccountID:   acctPayable.ID,
			AccountCode: acctPayable.Code,
			Debit:       amountCents,
			Description: "Abono a cuentas por pagar",
		},
		{
			AccountID:   acctCash.ID,
			AccountCode: acctCash.Code,
			Credit:      amountCents,
			Description: "Pago a proveedor",
		},
	}

	period, err := getOrCreatePeriod(ctx, h.periods, ev.CompanyID, ev.PaymentDate)
	if err != nil {
		return err
	}
	if period.Status == domain.PeriodClosed {
		h.log.Warn("período cerrado, asiento omitido", "payment_id", ev.PaymentID)
		return nil
	}

	seq, err := h.journals.NextVoucherSeq(ctx, ev.CompanyID, domain.VoucherExpense, ev.PaymentDate.Year())
	if err != nil {
		return fmt.Errorf("asignar comprobante: %w", err)
	}

	_, err = h.journals.Create(ctx, domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               ev.PaymentDate,
		Description:        "Pago a proveedor orden " + ev.PurchaseNumber,
		Status:             domain.StatusPosted,
		Source:             "purchase",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherExpense,
		VoucherNumber:      fmt.Sprintf("%s-%d-%05d", domain.VoucherExpense, ev.PaymentDate.Year(), seq),
		SourceDocumentID:   ev.PurchaseID,
		SourceDocumentType: "COMPRA_PAGO",
		Book:               domain.BookBoth,
		Lines:              lines,
	})
	return err
}
