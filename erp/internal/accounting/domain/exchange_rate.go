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
	Description  string
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

// TRMFetcher consulta la TRM oficial vigente (la que fija la Superintendencia Financiera, un
// solo valor por día) desde una fuente externa — implementado en infrastructure, nunca en
// application/domain, para no acoplar el dominio a detalles de HTTP. Deliberadamente NO es
// "compra/venta" comercial (esa es otra tasa, con spread, que cambia todo el día y no es la que
// exige la norma colombiana para contabilizar transacciones en moneda extranjera).
type TRMFetcher interface {
	// FetchTRM consulta la TRM oficial para la fecha exacta dada (siempre se pasa una fecha
	// concreta — nunca se le deja "adivinar hoy" a la fuente externa, así el día que aplica
	// nunca depende de en qué huso horario corra el servicio remoto).
	FetchTRM(ctx context.Context, date time.Time) (rate float64, description string, err error)
}
