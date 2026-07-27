package cartera

import "errors"

var (
	ErrLineAlreadyReconciled = errors.New("la línea ya tiene una marca de conciliación")
	ErrMarkNotFound          = errors.New("marca de conciliación no encontrada")
	ErrSameLineReconciliation = errors.New("no se puede conciliar una línea consigo misma")
)
