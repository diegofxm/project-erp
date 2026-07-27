package periods

import "errors"

var (
	ErrPeriodNotFound = errors.New("periods: periodo no encontrado")
	// ErrPeriodClosed: se intenta registrar un asiento en un periodo ya cerrado.
	ErrPeriodClosed = errors.New("periods: el periodo contable está cerrado")
	ErrPeriodAlreadyOpen = errors.New("periods: ya existe un periodo abierto para ese mes")
)
