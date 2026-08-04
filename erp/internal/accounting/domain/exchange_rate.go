package domain

import (
	"context"
	"errors"
	"time"
)

// ExchangeRate es la Tasa Representativa del Mercado (TRM) para un par de monedas en una fecha
// — no está ligada a una empresa (es un dato de mercado, igual para todas). RateX10000 guarda la
// tasa multiplicada por 10000 para evitar imprecisión de punto flotante (ej. TRM 4123.4567 se
// guarda como 41234567).
type ExchangeRate struct {
	RateDate     time.Time
	FromCurrency string
	ToCurrency   string
	RateX10000   int64
	Source       string
	CreatedAt    time.Time
}

// Rate devuelve la tasa como decimal (ej. 4123.4567).
func (r ExchangeRate) Rate() float64 {
	return float64(r.RateX10000) / 10000
}

var ErrExchangeRateNotFound = errors.New("no hay tasa de cambio registrada para esa fecha y par de monedas")

type ExchangeRateRepository interface {
	// Set crea o actualiza la tasa del día para ese par de monedas (upsert por rate_date+from+to).
	Set(ctx context.Context, r ExchangeRate) (*ExchangeRate, error)
	Get(ctx context.Context, date time.Time, from, to string) (*ExchangeRate, error)
	List(ctx context.Context, from, to time.Time) ([]ExchangeRate, error)
}
