package application

import (
	"context"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// RunTRMDailySync corre en segundo plano y sincroniza la TRM oficial todos los días a la 1:00
// a.m. hora Colombia — pensado para que el panel casi nunca necesite tocar el botón
// "Sincronizar" manual (que además se deshabilita solo del lado del frontend en cuanto ya exista
// la TRM de hoy). Si falla (servicio caído, sin red) se reintenta una sola vez una hora después
// en vez de esperar hasta el día siguiente — así una caída puntual no deja el día entero sin TRM,
// pero tampoco se insiste indefinidamente contra el servicio externo.
//
// onResult se llama después de cada intento (éxito o error) — pensado para loguear desde
// cmd/server/main.go sin que este paquete dependa de un logger concreto.
func RunTRMDailySync(ctx context.Context, uc *ExchangeRateUseCase, onResult func(rate *domain.ExchangeRate, err error)) {
	for {
		wait := untilNextRunAt(time.Now().In(bogota), 1, 0)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}

		rate, err := uc.Sync(ctx)
		if onResult != nil {
			onResult(rate, err)
		}
		if err == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
		}
		rate, err = uc.Sync(ctx)
		if onResult != nil {
			onResult(rate, err)
		}
	}
}

// untilNextRunAt calcula cuánto falta para la próxima vez que sean hour:minute en el huso de
// `now` — si esa hora ya pasó hoy, devuelve la de mañana.
func untilNextRunAt(now time.Time, hour, minute int) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}
