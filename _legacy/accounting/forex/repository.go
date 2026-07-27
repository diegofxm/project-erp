package forex

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del módulo de moneda extranjera.
type Repository interface {
	// SetRate registra o actualiza la tasa de cambio para una fecha y moneda.
	// Hace upsert por (rate_date, from_currency, to_currency).
	SetRate(ctx context.Context, req SetRateRequest) (*ExchangeRate, error)

	// GetRate devuelve la tasa vigente para una fecha.
	// Si no existe una tasa exacta para la fecha, retrocede hasta encontrar
	// la más reciente anterior (walk-back). Devuelve ErrRateNotFound si no hay ninguna.
	GetRate(ctx context.Context, date time.Time, fromCurrency string) (*ExchangeRate, error)

	// ListRates devuelve el historial de tasas para una moneda en un rango de fechas,
	// ordenado de más reciente a más antiguo.
	ListRates(ctx context.Context, from, to time.Time, fromCurrency string) ([]*ExchangeRate, error)

	// RevaluationBalances devuelve las cuentas con saldo neto no nulo en la moneda
	// indicada, junto con su saldo en COP registrado. Usa solo asientos POSTED.
	RevaluationBalances(ctx context.Context, companyID uuid.UUID, currency string) ([]revalBalance, error)
}
