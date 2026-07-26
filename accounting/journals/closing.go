package journals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/accounting/periods"
	"github.com/google/uuid"
)

const (
	profitAccountCode = "3605"
	lossAccountCode   = "3610"
)

var ErrNoPLActivity = errors.New("journals: no hay movimiento en cuentas de P&G para el año indicado")

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// CloseYear genera el asiento de cierre contable para el año dado y luego cierra
// todos los periodos del año. Lleva a cero todas las cuentas de Ingresos, Gastos
// y Costos, y registra la diferencia en 3605 (utilidad) o 3610 (pérdida).
//
// El período de diciembre debe estar OPEN. Tras el asiento se cierran todos los
// periodos del año automáticamente.
func (s *Service) CloseYear(ctx context.Context, companyID uuid.UUID, year int) (*JournalEntry, error) {
	plBalances, err := s.repo.GetYearPLBalances(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("cierre %d: obtener saldos P&G: %w", year, err)
	}
	if len(plBalances) == 0 {
		return nil, ErrNoPLActivity
	}

	// Ingresos: saldo acreedor (Balance < 0) → Debit = -Balance para llevarlo a cero.
	// Gastos/Costos: saldo deudor (Balance > 0) → Credit = Balance para llevarlo a cero.
	var lines []LineRequest
	var totalDebit, totalCredit int64

	for _, b := range plBalances {
		if b.Balance < 0 {
			amt := -b.Balance
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Debit:       amt,
				Description: "Cierre de ingresos " + fmt.Sprint(year),
			})
			totalDebit += amt
		} else if b.Balance > 0 {
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Credit:      b.Balance,
				Description: "Cierre de gastos/costos " + fmt.Sprint(year),
			})
			totalCredit += b.Balance
		}
	}

	net := totalDebit - totalCredit
	switch {
	case net > 0:
		lines = append(lines, LineRequest{
			AccountCode: profitAccountCode,
			Credit:      net,
			Description: fmt.Sprintf("Utilidad del ejercicio %d", year),
		})
	case net < 0:
		lines = append(lines, LineRequest{
			AccountCode: lossAccountCode,
			Debit:       absInt64(net),
			Description: fmt.Sprintf("Pérdida del ejercicio %d", year),
		})
	}

	decDate := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
	entry, err := s.Post(ctx, PostRequest{
		CompanyID:   companyID,
		Date:        decDate,
		Description: fmt.Sprintf("Asiento de cierre del ejercicio %d", year),
		Source:      "closing",
		EntryType:   EntryClosing,
		Lines:       lines,
	})
	if err != nil {
		return nil, fmt.Errorf("cierre %d: registrar asiento: %w", year, err)
	}

	if err := s.periodsSvc.CloseYear(ctx, companyID, year); err != nil {
		return nil, fmt.Errorf("cierre %d: cerrar periodos: %w", year, err)
	}

	return entry, nil
}

// OpenYear genera el asiento de apertura para el año indicado tomando el balance
// general al 31-dic del año anterior. Débita activos (saldo deudor) y acredita
// pasivos/patrimonio (saldo acreedor).
func (s *Service) OpenYear(ctx context.Context, companyID uuid.UUID, year int) (*JournalEntry, error) {
	asOf := time.Date(year-1, 12, 31, 23, 59, 59, 999999999, time.UTC)
	bsBalances, err := s.repo.GetBSBalances(ctx, companyID, asOf)
	if err != nil {
		return nil, fmt.Errorf("apertura %d: obtener balance general: %w", year, err)
	}
	if len(bsBalances) == 0 {
		return nil, fmt.Errorf("apertura %d: no hay saldos de balance general al 31-dic-%d", year, year-1)
	}

	var lines []LineRequest
	for _, b := range bsBalances {
		switch {
		case b.Balance > 0:
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Debit:       b.Balance,
				Description: "Apertura " + fmt.Sprint(year),
			})
		case b.Balance < 0:
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Credit:      -b.Balance,
				Description: "Apertura " + fmt.Sprint(year),
			})
		}
	}

	if len(lines) < 2 {
		return nil, fmt.Errorf("apertura %d: balance insuficiente para generar el asiento", year)
	}

	janDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	p, err := s.periodsSvc.GetOrCreate(ctx, companyID, janDate)
	if err != nil {
		return nil, fmt.Errorf("apertura %d: obtener periodo enero: %w", year, err)
	}
	if p.Status == periods.StatusClosed {
		return nil, periods.ErrPeriodClosed
	}

	return s.Post(ctx, PostRequest{
		CompanyID:   companyID,
		Date:        janDate,
		Description: fmt.Sprintf("Asiento de apertura del ejercicio %d", year),
		Source:      "opening",
		EntryType:   EntryOpening,
		Lines:       lines,
	})
}
