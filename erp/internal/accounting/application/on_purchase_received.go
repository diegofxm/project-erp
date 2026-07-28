package application

import (
	"context"
	"log"

	"github.com/diegofxm/erp/internal/accounting/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPurchaseReceived registra el asiento contable de la compra recibida.
//
// PUC colombiano:
//   - DB 143505 Inventario de mercancías  (subtotal sin IVA)
//   - DB 240810 IVA por descontar         (IVA soportado, si aplica)
//   - CR 220505 Proveedores nacionales    (total = subtotal + IVA)
type OnPurchaseReceived struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
}

func NewOnPurchaseReceived(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *OnPurchaseReceived {
	return &OnPurchaseReceived{accounts: accounts, periods: periods, journals: journals}
}

func (h *OnPurchaseReceived) Register(bus *events.Bus) {
	bus.Subscribe(purchasedomain.PurchaseReceived{}.EventName(), func(evt events.Event) {
		ev, ok := evt.(purchasedomain.PurchaseReceived)
		if !ok {
			return
		}
		ctx := context.Background()
		if err := h.handle(ctx, ev); err != nil {
			log.Printf("accounting: asiento compra %s: %v", ev.PurchaseID, err)
		}
	})
}

func (h *OnPurchaseReceived) handle(ctx context.Context, ev purchasedomain.PurchaseReceived) error {
	totalCents := toCents(ev.Total)
	taxCents := toCents(ev.TaxAmount)
	subtotalCents := totalCents - taxCents

	acctInventory, err := h.accounts.GetPostable(ctx, "143505")
	if err != nil {
		return err
	}
	acctSupplier, err := h.accounts.GetPostable(ctx, "220505")
	if err != nil {
		return err
	}

	lines := []*domain.JournalLine{
		{
			AccountID:   acctInventory.ID,
			AccountCode: acctInventory.Code,
			Debit:       subtotalCents,
			Description: "Compra de mercancía",
		},
		{
			AccountID:   acctSupplier.ID,
			AccountCode: acctSupplier.Code,
			Credit:      totalCents,
			Description: "Proveedor por pagar",
		},
	}

	if taxCents > 0 {
		acctIVA, err := h.accounts.GetPostable(ctx, "240810")
		if err != nil {
			return err
		}
		lines = append(lines, &domain.JournalLine{
			AccountID:   acctIVA.ID,
			AccountCode: acctIVA.Code,
			Debit:       taxCents,
			Description: "IVA descontable",
		})
	}

	period, err := getOrCreatePeriod(ctx, h.periods, ev.CompanyID, ev.IssueDate)
	if err != nil {
		return err
	}
	if period.Status == domain.PeriodClosed {
		log.Printf("accounting: período cerrado para compra %s — asiento omitido", ev.PurchaseID)
		return nil
	}

	_, err = h.journals.Create(ctx, domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               ev.IssueDate,
		Description:        "Compra " + ev.PurchaseID.String(),
		Status:             domain.StatusPosted,
		Source:             "purchase",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherExpense,
		SourceDocumentID:   ev.PurchaseID,
		SourceDocumentType: "COMPRA",
		Book:               domain.BookBoth,
		Lines:              lines,
	})
	return err
}
