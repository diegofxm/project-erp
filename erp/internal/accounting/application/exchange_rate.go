package application

import (
	"context"
	"fmt"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// ExchangeRateUseCase administra la TRM (Tasa Representativa del Mercado) diaria — no es un dato
// por empresa, es un dato de mercado que todas comparten. fetcher puede ser nil si no hay
// integración externa configurada — Sync devuelve error en ese caso, pero Set (captura manual)
// sigue funcionando igual, sin depender de él.
type ExchangeRateUseCase struct {
	rates   domain.ExchangeRateRepository
	fetcher domain.TRMFetcher
}

func NewExchangeRateUseCase(rates domain.ExchangeRateRepository, fetcher domain.TRMFetcher) *ExchangeRateUseCase {
	return &ExchangeRateUseCase{rates: rates, fetcher: fetcher}
}

type SetExchangeRateRequest struct {
	RateDate     time.Time
	FromCurrency string
	ToCurrency   string
	Rate         float64 // ej. 4123.4567 pesos por 1 USD
	Source       string
}

func (uc *ExchangeRateUseCase) Set(ctx context.Context, req SetExchangeRateRequest) (*domain.ExchangeRate, error) {
	if req.FromCurrency == "" {
		return nil, fmt.Errorf("from_currency es obligatorio")
	}
	if req.ToCurrency == "" {
		req.ToCurrency = "COP"
	}
	if req.Rate <= 0 {
		return nil, fmt.Errorf("rate debe ser mayor que cero")
	}
	source := req.Source
	if source == "" {
		source = "MANUAL"
	}
	return uc.rates.Set(ctx, domain.ExchangeRate{
		RateDate: req.RateDate, FromCurrency: req.FromCurrency, ToCurrency: req.ToCurrency,
		RateX10000: int64(req.Rate * 10000), Source: source,
	})
}

func (uc *ExchangeRateUseCase) Get(ctx context.Context, date time.Time, from, to string) (*domain.ExchangeRate, error) {
	if to == "" {
		to = "COP"
	}
	return uc.rates.Get(ctx, date, from, to)
}

func (uc *ExchangeRateUseCase) List(ctx context.Context, from, to time.Time) ([]domain.ExchangeRate, error) {
	return uc.rates.List(ctx, from, to)
}

// Sync consulta la TRM oficial vigente (Superfinanciera, vía dolarapi.com) y la guarda como tasa
// del día para USD→COP. Si la fuente externa falla (sin red, API caída), el error se propaga tal
// cual y la captura manual (Set) sigue disponible como respaldo — no hay estado a limpiar.
func (uc *ExchangeRateUseCase) Sync(ctx context.Context) (*domain.ExchangeRate, error) {
	if uc.fetcher == nil {
		return nil, fmt.Errorf("sincronización de TRM no configurada")
	}
	rate, date, err := uc.fetcher.FetchTRM(ctx)
	if err != nil {
		return nil, fmt.Errorf("consultar TRM: %w", err)
	}
	return uc.rates.Set(ctx, domain.ExchangeRate{
		RateDate: date, FromCurrency: "USD", ToCurrency: "COP",
		RateX10000: int64(rate * 10000), Source: "DOLARAPI",
	})
}
