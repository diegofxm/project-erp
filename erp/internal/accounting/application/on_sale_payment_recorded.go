package application

import (
	"context"
	"log"

	"github.com/diegofxm/erp/internal/accounting/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnSalePaymentRecorded registra el asiento contable cuando se recibe un pago de cliente.
//
// PUC colombiano:
//   - DB 110505 Caja general / 111005 Bancos (según medio de pago)
//   - CR 130505 Clientes nacionales (abono a cartera)
type OnSalePaymentRecorded struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
}

func NewOnSalePaymentRecorded(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *OnSalePaymentRecorded {
	return &OnSalePaymentRecorded{accounts: accounts, periods: periods, journals: journals}
}

func (h *OnSalePaymentRecorded) Register(bus *events.Bus) {
	bus.Subscribe(salesdomain.SalePaymentRecorded{}.EventName(), func(evt events.Event) {
		ev, ok := evt.(salesdomain.SalePaymentRecorded)
		if !ok {
			return
		}
		ctx := context.Background()
		if err := h.handle(ctx, ev); err != nil {
			log.Printf("accounting: asiento pago venta %s: %v", ev.PaymentID, err)
		}
	})
}

func (h *OnSalePaymentRecorded) handle(ctx context.Context, ev salesdomain.SalePaymentRecorded) error {
	amountCents := toCents(ev.Amount)

	acctCash, err := h.accounts.GetPostable(ctx, cashAccountCode(string(ev.PaymentMethod)))
	if err != nil {
		return err
	}
	acctReceivable, err := h.accounts.GetPostable(ctx, "130505")
	if err != nil {
		return err
	}

	lines := []*domain.JournalLine{
		{
			AccountID:   acctCash.ID,
			AccountCode: acctCash.Code,
			Debit:       amountCents,
			Description: "Pago recibido de cliente",
		},
		{
			AccountID:   acctReceivable.ID,
			AccountCode: acctReceivable.Code,
			Credit:      amountCents,
			Description: "Abono a cartera",
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

	_, err = h.journals.Create(ctx, domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               ev.PaymentDate,
		Description:        "Pago recibido " + ev.PaymentID.String(),
		Status:             domain.StatusPosted,
		Source:             "sales",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherIncome,
		SourceDocumentID:   ev.SaleID,
		SourceDocumentType: "VENTA_PAGO",
		Book:               domain.BookBoth,
		Lines:              lines,
	})
	return err
}

// cashAccountCode resuelve la cuenta de caja/bancos según el medio de pago —
// compartida por los listeners de pago de ventas y de compras.
func cashAccountCode(method string) string {
	if method == "cash" {
		return "110505" // Caja general
	}
	return "111005" // Bancos, moneda nacional
}
