package application

import (
	"context"
	"fmt"
	"log"

	"github.com/diegofxm/erp/internal/accounting/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPurchaseReceived registra el asiento contable de la compra recibida.
//
// PUC colombiano:
//   - DB 143505 Mercancías no fabricadas por la empresa (subtotal sin IVA)
//   - DB 2408   Impuesto sobre las ventas por pagar     (IVA soportado/descontable, si aplica —
//     misma cuenta que usa on_sale_confirmed para el IVA generado; el saldo neto de 2408
//     ya refleja "generado - descontable")
//   - CR 2205   Proveedores nacionales    (total - retenciones = lo que realmente se le paga)
//   - CR <cuenta del concepto>            (una línea por cada retención aplicada, pasivo con la DIAN)
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

	var withholdingCents int64
	for _, w := range ev.Withholdings {
		withholdingCents += toCents(w.Amount)
	}
	acctSupplier, err := h.accounts.GetPostable(ctx, "2205")
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
			Credit:      totalCents - withholdingCents,
			Description: "Proveedor por pagar",
		},
	}

	if taxCents > 0 {
		acctIVA, err := h.accounts.GetPostable(ctx, "2408")
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

	for _, w := range ev.Withholdings {
		acctWh, err := h.accounts.GetPostable(ctx, w.AccountPayable)
		if err != nil {
			return err
		}
		lines = append(lines, &domain.JournalLine{
			AccountID:   acctWh.ID,
			AccountCode: acctWh.Code,
			Credit:      toCents(w.Amount),
			Description: "Retención " + w.ConceptName,
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

	seq, err := h.journals.NextVoucherSeq(ctx, ev.CompanyID, domain.VoucherExpense, ev.IssueDate.Year())
	if err != nil {
		return fmt.Errorf("asignar comprobante: %w", err)
	}

	_, err = h.journals.Create(ctx, domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               ev.IssueDate,
		Description:        "Compra " + ev.Number,
		Status:             domain.StatusPosted,
		Source:             "purchase",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherExpense,
		VoucherNumber:      fmt.Sprintf("%s-%d-%05d", domain.VoucherExpense, ev.IssueDate.Year(), seq),
		SourceDocumentID:   ev.PurchaseID,
		SourceDocumentType: "COMPRA",
		Book:               domain.BookBoth,
		Lines:              lines,
	})
	return err
}
