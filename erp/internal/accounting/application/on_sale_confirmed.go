package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/diegofxm/erp/internal/accounting/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnSaleConfirmed suscribe el bus de eventos y registra el asiento contable
// automático cuando se confirma una venta.
//
// Cuentas PUC colombiano:
//   - 130505 Clientes nacionales      (débito = total de la venta)
//   - 413595 Venta de otros productos (crédito = subtotal sin IVA)
//   - 2408   Impuesto sobre las ventas por pagar (crédito = IVA generado; misma cuenta que
//     on_purchase_received usa para el IVA descontable — el saldo neto de 2408 ya refleja
//     "generado - descontable")
type OnSaleConfirmed struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
	log      *slog.Logger
}

func NewOnSaleConfirmed(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
	log *slog.Logger,
) *OnSaleConfirmed {
	return &OnSaleConfirmed{accounts: accounts, periods: periods, journals: journals, log: log}
}

func (h *OnSaleConfirmed) Register(bus *events.Bus) {
	bus.Subscribe(salesdomain.SaleConfirmed{}.EventName(), func(ctx context.Context, evt events.Event) error {
		ev, ok := evt.(salesdomain.SaleConfirmed)
		if !ok {
			return nil
		}
		if err := h.handle(ctx, ev); err != nil {
			h.log.Error("asiento venta", "sale_id", ev.SaleID, "error", err)
			return fmt.Errorf("accounting: asiento venta %s: %w", ev.SaleID, err)
		}
		return nil
	})
}

func (h *OnSaleConfirmed) handle(ctx context.Context, ev salesdomain.SaleConfirmed) error {
	// Convertir float64 a centavos enteros (redondeo bancario)
	totalCents := toCents(ev.Total)
	taxCents := toCents(ev.TaxAmount)
	subtotalCents := totalCents - taxCents

	// Resolver cuentas (validadas como posteables)
	acctReceivable, err := h.accounts.GetPostable(ctx, "130505")
	if err != nil {
		return err
	}
	acctIncome, err := h.accounts.GetPostable(ctx, "413595")
	if err != nil {
		return err
	}

	lines := []*domain.JournalLine{
		{
			AccountID:   acctReceivable.ID,
			AccountCode: acctReceivable.Code,
			Debit:       totalCents,
			Description: "Venta confirmada",
		},
		{
			AccountID:   acctIncome.ID,
			AccountCode: acctIncome.Code,
			Credit:      subtotalCents,
			Description: "Ingreso por venta",
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
			Credit:      taxCents,
			Description: "IVA generado",
		})
	}

	period, err := getOrCreatePeriod(ctx, h.periods, ev.CompanyID, ev.IssueDate)
	if err != nil {
		return err
	}
	if period.Status == domain.PeriodClosed {
		h.log.Warn("período cerrado, asiento omitido", "sale_id", ev.SaleID)
		return nil
	}

	entry := domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               ev.IssueDate,
		Description:        "Venta " + ev.Number,
		Status:             domain.StatusPosted,
		Source:             "sales",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherIncome,
		SourceDocumentID:   ev.SaleID,
		SourceDocumentType: "VENTA",
		Book:               domain.BookBoth,
		Lines:              lines,
	}

	seq, err := h.journals.NextVoucherSeq(ctx, ev.CompanyID, domain.VoucherIncome, ev.IssueDate.Year())
	if err != nil {
		return fmt.Errorf("asignar comprobante: %w", err)
	}
	entry.VoucherNumber = fmt.Sprintf("%s-%d-%05d", domain.VoucherIncome, ev.IssueDate.Year(), seq)

	if _, err := h.journals.Create(ctx, entry); err != nil {
		return err
	}
	return nil
}

func toCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}
