package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// bogota es el huso horario que decide "qué día es hoy" para la TRM — la Superfinanciera fija un
// solo valor oficial por día calendario colombiano, sin importar en qué huso corra el servidor.
var bogota = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return time.FixedZone("America/Bogota", -5*60*60)
	}
	return loc
}()

// ExchangeRateUseCase administra la TRM (Tasa Representativa del Mercado) diaria — no es un dato
// por empresa, es un dato de mercado que todas comparten. fetcher puede ser nil si no hay
// integración externa configurada — Sync/GetOrFetch devuelven error en ese caso, pero Set (captura
// manual) sigue funcionando igual, sin depender de él.
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
	Description  string
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
	description := req.Description
	if description == "" && source == "MANUAL" {
		description = "Editado manualmente"
	}
	return uc.rates.Set(ctx, domain.ExchangeRate{
		RateDate: req.RateDate, FromCurrency: req.FromCurrency, ToCurrency: req.ToCurrency,
		RateX10000: int64(req.Rate * 10000), Source: source, Description: description,
	})
}

func (uc *ExchangeRateUseCase) Get(ctx context.Context, date time.Time, from, to string) (*domain.ExchangeRate, error) {
	if to == "" {
		to = "COP"
	}
	return uc.rates.Get(ctx, date, from, to)
}

// List devuelve una página de tasas (más reciente primero) y el total de filas — para paginar en
// vez de traer todo el historial de una sola vez.
func (uc *ExchangeRateUseCase) List(ctx context.Context, limit, offset int) ([]domain.ExchangeRate, int, error) {
	return uc.rates.List(ctx, limit, offset)
}

// GetToday es un atajo de solo-lectura contra la base de datos (nunca toca el servicio externo)
// para mostrar la TRM de hoy junto al título del panel, sin importar en qué página de la lista
// esté parado el usuario ni si hoy ya se sincronizó.
func (uc *ExchangeRateUseCase) GetToday(ctx context.Context) (*domain.ExchangeRate, error) {
	today := time.Now().In(bogota)
	date := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return uc.Get(ctx, date, "USD", "COP")
}

// Sync consulta la TRM oficial vigente de HOY (hora Colombia) en el servicio propio (ver
// infrastructure/trmapi) y la guarda como tasa del día para USD→COP. Si la fuente externa falla
// (sin red, servicio caído), el error se propaga tal cual y la captura manual (Set) sigue
// disponible como respaldo — no hay estado a limpiar.
func (uc *ExchangeRateUseCase) Sync(ctx context.Context) (*domain.ExchangeRate, error) {
	today := time.Now().In(bogota)
	date := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	return uc.syncDate(ctx, date)
}

// GetOrFetch busca la TRM de una fecha específica en la base local primero — si ya está, no toca
// el servicio externo. Si no está, la consulta una única vez al servicio TRM propio y la guarda
// (la TRM histórica no cambia, así que nunca vuelve a pedirse esa misma fecha). Pensada para que
// el contador busque cualquier fecha pasada sin depender de que el disparador diario ya haya
// pasado por ahí.
func (uc *ExchangeRateUseCase) GetOrFetch(ctx context.Context, date time.Time, from, to string) (*domain.ExchangeRate, error) {
	if from == "" {
		from = "USD"
	}
	if to == "" {
		to = "COP"
	}
	existing, err := uc.rates.Get(ctx, date, from, to)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrExchangeRateNotFound) {
		return nil, err
	}
	return uc.syncDate(ctx, date)
}

func (uc *ExchangeRateUseCase) syncDate(ctx context.Context, date time.Time) (*domain.ExchangeRate, error) {
	if uc.fetcher == nil {
		return nil, fmt.Errorf("sincronización de TRM no configurada")
	}
	rate, description, err := uc.fetcher.FetchTRM(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("consultar TRM: %w", err)
	}
	return uc.rates.Set(ctx, domain.ExchangeRate{
		RateDate: date, FromCurrency: "USD", ToCurrency: "COP",
		RateX10000: int64(rate * 10000), Source: "SUPERFINANCIERA", Description: description,
	})
}
