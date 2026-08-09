package application

import (
	"context"
	"fmt"
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
		runOnceSafely(ctx, uc, onResult)
	}
}

// runOnceSafely ejecuta un intento de sincronización (con su reintento a la hora) protegido con
// recover(). Es la única goroutine de larga vida del servidor -- sin este recover, un panic acá
// (ej. nil pointer en el cliente TRM) tumbaría TODO el proceso del servidor, no solo esta tarea,
// porque un panic sin recuperar en cualquier goroutine termina el programa completo. Al envolver
// solo el intento (no todo RunTRMDailySync), un panic aislado no mata tampoco el scheduler para
// los días siguientes -- el for de arriba sigue esperando al próximo 1:00 a.m. con normalidad.
func runOnceSafely(ctx context.Context, uc *ExchangeRateUseCase, onResult func(rate *domain.ExchangeRate, err error)) {
	defer func() {
		if rec := recover(); rec != nil && onResult != nil {
			onResult(nil, fmt.Errorf("panic recuperado en sincronización TRM: %v", rec))
		}
	}()

	rate, err := uc.Sync(ctx)
	if onResult != nil {
		onResult(rate, err)
	}
	if err == nil {
		return
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

// untilNextRunAt calcula cuánto falta para la próxima vez que sean hour:minute en el huso de
// `now` — si esa hora ya pasó hoy, devuelve la de mañana.
func untilNextRunAt(now time.Time, hour, minute int) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next.Sub(now)
}
