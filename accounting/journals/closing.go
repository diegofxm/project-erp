package journals

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/diegofxm/accounting/periods"
	"github.com/google/uuid"
)

const (
	// Cuenta PUC donde va la utilidad neta al cierre del ejercicio.
	profitAccountCode = "3605"
	// Cuenta PUC donde va la pérdida neta al cierre del ejercicio.
	lossAccountCode = "3610"
)

var ErrNoPLActivity = errors.New("journals: no hay movimiento en cuentas de P&G para el año indicado")

// CloseYear genera el asiento de cierre contable para el año dado y luego cierra
// todos los periodos del año. El asiento lleva a cero todas las cuentas de
// Ingresos, Gastos y Costos, y registra la utilidad (3605) o pérdida (3610) neta
// en el Patrimonio.
//
// El período de diciembre debe estar OPEN al momento de llamar esta función.
// Tras el asiento de cierre se cierran todos los periodos del año automáticamente.
func (s *Service) CloseYear(ctx context.Context, companyID uuid.UUID, year int) (*JournalEntry, error) {
	plBalances, err := s.repo.GetYearPLBalances(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("cierre %d: obtener saldos P&G: %w", year, err)
	}
	if len(plBalances) == 0 {
		return nil, ErrNoPLActivity
	}

	// Construir líneas de cierre.
	// - Ingresos: saldo acreedor (Balance < 0) → Debit = -Balance para llevarlo a cero.
	// - Gastos/Costos: saldo deudor (Balance > 0) → Credit = Balance para llevarlo a cero.
	var lines []LineRequest
	var totalDebit, totalCredit float64

	for _, b := range plBalances {
		if b.Balance < 0 {
			// Cuenta de ingresos con saldo acreedor — cerramos con débito.
			amt := math.Abs(b.Balance)
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Debit:       amt,
				Description: "Cierre de ingresos " + fmt.Sprint(year),
			})
			totalDebit += amt
		} else if b.Balance > 0 {
			// Cuenta de gastos/costos con saldo deudor — cerramos con crédito.
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Credit:      b.Balance,
				Description: "Cierre de gastos/costos " + fmt.Sprint(year),
			})
			totalCredit += b.Balance
		}
	}

	// La diferencia va a utilidad o pérdida del ejercicio.
	net := totalDebit - totalCredit
	switch {
	case net > 0.01:
		// Utilidad: crédito 3605
		lines = append(lines, LineRequest{
			AccountCode: profitAccountCode,
			Credit:      net,
			Description: fmt.Sprintf("Utilidad del ejercicio %d", year),
		})
	case net < -0.01:
		// Pérdida: débito 3610
		lines = append(lines, LineRequest{
			AccountCode: lossAccountCode,
			Debit:       math.Abs(net),
			Description: fmt.Sprintf("Pérdida del ejercicio %d", year),
		})
	}

	// Registrar el asiento en diciembre del año de cierre.
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

	// Cerrar todos los periodos abiertos del año.
	if err := s.periodsSvc.CloseYear(ctx, companyID, year); err != nil {
		return nil, fmt.Errorf("cierre %d: cerrar periodos: %w", year, err)
	}

	return entry, nil
}

// OpenYear genera el asiento de apertura para el año indicado, tomando como base
// el balance general al 31-dic del año anterior. Crea (si no existe) el período
// de enero del año nuevo.
//
// Débita cada cuenta de Activo con saldo positivo y acredita cada cuenta de
// Pasivo y Patrimonio con saldo negativo (acreedor), reproduciendo el balance
// inicial del nuevo ejercicio.
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
		case b.Balance > 0.01:
			// Saldo deudor (activos): débito de apertura.
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Debit:       b.Balance,
				Description: "Apertura " + fmt.Sprint(year),
			})
		case b.Balance < -0.01:
			// Saldo acreedor (pasivos y patrimonio): crédito de apertura.
			lines = append(lines, LineRequest{
				AccountCode: b.AccountCode,
				Credit:      math.Abs(b.Balance),
				Description: "Apertura " + fmt.Sprint(year),
			})
		}
	}

	if len(lines) < 2 {
		return nil, fmt.Errorf("apertura %d: balance insuficiente para generar el asiento", year)
	}

	// Verificar que el período de enero esté abierto (o crear si no existe).
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
