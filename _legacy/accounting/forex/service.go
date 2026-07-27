package forex

import (
	"context"
	"fmt"
	"time"

	"github.com/diegofxm/accounting/journals"
	"github.com/google/uuid"
)

const (
	// AccountForexGain es la cuenta PUC de ingreso por diferencia en cambio favorable.
	// Valor por defecto: 4210 Financieros (no operacionales). Configurable vía Service.GainAccount.
	AccountForexGain = "4210"
	// AccountForexLoss es la cuenta PUC de gasto por diferencia en cambio desfavorable.
	// Valor por defecto: 5306 Diferencia en cambio (gastos financieros). Configurable vía Service.LossAccount.
	AccountForexLoss = "5306"
)

// Service expone las operaciones de moneda extranjera: tasas, conversión y revaluación.
type Service struct {
	repo        Repository
	journals    *journals.Service
	GainAccount string // cuenta PUC de ingreso por diferencia en cambio (default: AccountForexGain)
	LossAccount string // cuenta PUC de gasto por diferencia en cambio (default: AccountForexLoss)
}

// NewService crea el servicio con las cuentas por defecto.
func NewService(repo Repository, journalsSvc *journals.Service) *Service {
	return &Service{
		repo:        repo,
		journals:    journalsSvc,
		GainAccount: AccountForexGain,
		LossAccount: AccountForexLoss,
	}
}

// SetRate registra o actualiza la tasa de cambio para una fecha y moneda.
func (s *Service) SetRate(ctx context.Context, req SetRateRequest) (*ExchangeRate, error) {
	return s.repo.SetRate(ctx, req)
}

// GetRate devuelve la tasa vigente para una fecha.
// Si no hay tasa exacta, retrocede hasta encontrar la más reciente anterior.
func (s *Service) GetRate(ctx context.Context, date time.Time, currency string) (*ExchangeRate, error) {
	return s.repo.GetRate(ctx, date, currency)
}

// ListRates devuelve el historial de tasas para una moneda en un rango de fechas.
func (s *Service) ListRates(ctx context.Context, from, to time.Time, currency string) ([]*ExchangeRate, error) {
	return s.repo.ListRates(ctx, from, to, currency)
}

// Convert convierte centavos de moneda extranjera a centavos COP en la fecha indicada.
func (s *Service) Convert(ctx context.Context, date time.Time, currency string, foreignCents int64) (int64, error) {
	rate, err := s.repo.GetRate(ctx, date, currency)
	if err != nil {
		return 0, fmt.Errorf("forex convert %s al %s: %w", currency, date.Format("2006-01-02"), err)
	}
	return rate.CopCents(foreignCents), nil
}

// Revalue calcula y registra el diferencial cambiario al cierre de un período.
//
// Para cada cuenta con saldo neto en moneda extranjera, calcula la diferencia entre
// el valor COP registrado y el valor al nuevo tipo de cambio (asOf). La ganancia neta
// se acredita a s.GainAccount (default 4210); la pérdida neta se debita a s.LossAccount
// (default 5306).
//
// Lógica de signos por cuenta:
//
//	Diff > 0 (activo aumentó o pasivo disminuyó): DR cuenta, CR 4210
//	Diff < 0 (activo disminuyó o pasivo aumentó): CR cuenta, DR 5306
//
// El asiento generado siempre cuadra porque TotalDiff = ΣDiff = DR_cuentas − CR_cuentas,
// y el offset (4210 o 5306) lo absorbe exactamente.
// Si TotalDiff = 0 las cuentas se compensan entre sí y no se genera offset ni asiento.
func (s *Service) Revalue(ctx context.Context, companyID uuid.UUID, currency string, asOf time.Time) (*RevaluationResult, error) {
	rate, err := s.repo.GetRate(ctx, asOf, currency)
	if err != nil {
		return nil, fmt.Errorf("forex revalue: tasa %s al %s: %w", currency, asOf.Format("2006-01-02"), err)
	}

	balances, err := s.repo.RevaluationBalances(ctx, companyID, currency)
	if err != nil {
		return nil, fmt.Errorf("forex revalue: saldos %s: %w", currency, err)
	}

	result := &RevaluationResult{
		Currency:   currency,
		AsOf:       asOf,
		RateX10000: rate.RateX10000,
	}

	for _, b := range balances {
		newCOP := rate.CopCents(b.Foreign)
		diff := newCOP - b.COP
		if diff == 0 {
			continue
		}
		result.Lines = append(result.Lines, RevaluationLine{
			AccountID:      b.AccountID,
			AccountCode:    b.AccountCode,
			AccountName:    b.AccountName,
			ForeignBalance: b.Foreign,
			RecordedCOP:    b.COP,
			NewCOP:         newCOP,
			Diff:           diff,
		})
		result.TotalDiff += diff
	}

	if len(result.Lines) == 0 {
		return result, nil // sin diferencias — no se genera asiento
	}

	// Construir líneas del asiento: una por cuenta afectada.
	monthLabel := asOf.Format("Jan 2006")
	lines := make([]journals.LineRequest, 0, len(result.Lines)+1)
	for _, l := range result.Lines {
		req := journals.LineRequest{
			AccountCode: l.AccountCode,
			Description: fmt.Sprintf("Dif. cambio %s %s", currency, monthLabel),
		}
		if l.Diff > 0 {
			req.Debit = l.Diff
		} else {
			req.Credit = -l.Diff
		}
		lines = append(lines, req)
	}

	// Línea de offset: ganancia → CR 4210 / pérdida → DR 5306.
	if result.TotalDiff > 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode: s.GainAccount,
			Credit:      result.TotalDiff,
			Description: fmt.Sprintf("Diferencia en cambio %s %s — ganancia", currency, monthLabel),
		})
	} else if result.TotalDiff < 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode: s.LossAccount,
			Debit:       -result.TotalDiff,
			Description: fmt.Sprintf("Diferencia en cambio %s %s — pérdida", currency, monthLabel),
		})
	}
	// TotalDiff == 0: las cuentas individuales se compensan; no se necesita offset.

	entry, err := s.journals.Post(ctx, journals.PostRequest{
		CompanyID:   companyID,
		Date:        asOf,
		Description: fmt.Sprintf("Diferencial cambiario %s — %s", currency, asOf.Format("enero 2006")),
		Source:      "forex_revaluation",
		EntryType:   journals.EntryAdjustment,
		VoucherType: "DC",
		Lines:       lines,
	})
	if err != nil {
		return nil, fmt.Errorf("forex revalue: registrar asiento: %w", err)
	}

	result.JournalEntryID = &entry.ID
	return result, nil
}
