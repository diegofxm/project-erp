package domain

import "errors"

var (
	ErrAbsenceNotFound   = errors.New("hr: ausencia no encontrada")
	ErrAbsenceNotPending = errors.New("hr: la ausencia no está en estado pendiente")
	ErrInvalidDateRange  = errors.New("hr: la fecha de fin debe ser posterior a la de inicio")
)
