package application

import (
	"context"
	"fmt"
	"log"

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
}

func NewOnPurchasePaymentRecorded(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *OnPurchasePaymentRecorded {
	return &OnPurchasePaymentRecorded{accounts: accounts, periods: periods, journals: journals}
}

func (h *OnPurchasePaymentRecorded) Register(bus *events.Bus) {
	bus.Subscribe(purchasedomain.PurchasePaymentRecorded{}.EventName(), func(evt events.Event) {
		ev, ok := evt.(purchasedomain.PurchasePaymentRecorded)
		if !ok {
			return
		}
		ctx := context.Background()
		if err := h.handle(ctx, ev); err != nil {
			log.Printf("accounting: asiento pago compra %s: %v", ev.PaymentID, err)
		}
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
		log.Printf("accounting: período cerrado para pago %s — asiento omitido", ev.PaymentID)
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
