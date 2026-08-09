package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
	payrolldomain "github.com/diegofxm/erp/internal/payroll/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPayrollGenerated registra el asiento de gasto de personal cuando se aprueba una nómina.
//
// PUC colombiano (simplificado):
//   - DB 510506 Sueldos y salarios        (devengado bruto)
//   - CR 236540 Retenciones y aportes     (deducciones empleado)
//   - CR 2505   Salarios por pagar        (neto a pagar)
type OnPayrollGenerated struct {
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
	log      *slog.Logger
}

func NewOnPayrollGenerated(
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
	log *slog.Logger,
) *OnPayrollGenerated {
	return &OnPayrollGenerated{accounts: accounts, periods: periods, journals: journals, log: log}
}

func (h *OnPayrollGenerated) Register(bus *events.Bus) {
	bus.Subscribe(payrolldomain.PayrollGenerated{}.EventName(), func(ctx context.Context, evt events.Event) error {
		ev, ok := evt.(payrolldomain.PayrollGenerated)
		if !ok {
			return nil
		}
		if err := h.handle(ctx, ev); err != nil {
			h.log.Error("asiento nómina", "payslip_id", ev.PayslipID, "error", err)
			return fmt.Errorf("accounting: asiento nómina %s: %w", ev.PayslipID, err)
		}
		return nil
	})
}

func (h *OnPayrollGenerated) handle(ctx context.Context, ev payrolldomain.PayrollGenerated) error {
	acctSalary, err := h.accounts.GetPostable(ctx, "510506")
	if err != nil {
		return err
	}
	acctDeductions, err := h.accounts.GetPostable(ctx, "236540")
	if err != nil {
		return err
	}
	acctPayable, err := h.accounts.GetPostable(ctx, "2505")
	if err != nil {
		return err
	}

	lines := []*domain.JournalLine{
		{
			AccountID:   acctSalary.ID,
			AccountCode: acctSalary.Code,
			Debit:       ev.TotalEarnedCents,
			Description: fmt.Sprintf("Nómina %d/%02d", ev.PeriodYear, ev.PeriodMonth),
		},
	}
	if ev.TotalDeductedCents > 0 {
		lines = append(lines, &domain.JournalLine{
			AccountID:   acctDeductions.ID,
			AccountCode: acctDeductions.Code,
			Credit:      ev.TotalDeductedCents,
			Description: "Deducciones empleado",
		})
	}
	lines = append(lines, &domain.JournalLine{
		AccountID:   acctPayable.ID,
		AccountCode: acctPayable.Code,
		Credit:      ev.NetPayCents,
		Description: "Neto a pagar empleado",
	})

	// Fecha del asiento: último día del mes de la nómina
	date := time.Date(ev.PeriodYear, time.Month(ev.PeriodMonth+1), 0, 0, 0, 0, 0, time.UTC)

	period, err := getOrCreatePeriod(ctx, h.periods, ev.CompanyID, date)
	if err != nil {
		return err
	}
	if period.Status == domain.PeriodClosed {
		h.log.Warn("período cerrado, asiento omitido", "payslip_id", ev.PayslipID)
		return nil
	}

	_, err = h.journals.Create(ctx, domain.JournalEntry{
		CompanyID:          ev.CompanyID,
		PeriodID:           period.ID,
		Date:               date,
		Description:        fmt.Sprintf("Nómina %d/%02d empleado %s", ev.PeriodYear, ev.PeriodMonth, ev.EmployeeID),
		Status:             domain.StatusPosted,
		Source:             "payroll",
		EntryType:          domain.EntryAutomatic,
		VoucherType:        domain.VoucherPayroll,
		SourceDocumentID:   ev.PayslipID,
		SourceDocumentType: "NOMINA",
		Book:               domain.BookBoth,
		Lines:              lines,
	})
	return err
}
