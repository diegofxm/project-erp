package journals

import "errors"

var (
	ErrJournalNotFound = errors.New("journals: asiento no encontrado")
	// ErrImbalancedEntry: la suma de débitos debe ser exactamente igual a la suma de créditos.
	ErrImbalancedEntry = errors.New("journals: el asiento no cuadra — suma de débitos ≠ suma de créditos")
	ErrEmptyLines      = errors.New("journals: el asiento debe tener al menos dos líneas")
	ErrJournalVoided   = errors.New("journals: el asiento ya fue anulado")
	// ErrInvalidLine: exactamente uno de debit o credit debe ser > 0 en cada línea.
	ErrInvalidLine = errors.New("journals: cada línea debe tener exactamente un valor > 0 (debit o credit)")
)
