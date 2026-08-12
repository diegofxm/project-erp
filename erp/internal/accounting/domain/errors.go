package domain

import "errors"

var (
	ErrAccountNotFound    = errors.New("cuenta contable no encontrada")
	ErrAccountNotPosting  = errors.New("la cuenta no permite imputación directa")
	ErrAccountInactive    = errors.New("la cuenta está inactiva")
	ErrPeriodNotFound     = errors.New("período contable no encontrado")
	ErrPeriodClosed       = errors.New("el período contable está cerrado")
	ErrJournalNotFound    = errors.New("asiento contable no encontrado")
	ErrJournalVoided      = errors.New("el asiento ya fue anulado")
	ErrEmptyLines         = errors.New("el asiento requiere al menos 2 líneas")
	ErrInvalidLine        = errors.New("cada línea debe tener exactamente débito XOR crédito > 0")
	ErrImbalancedEntry    = errors.New("partida no cuadrada: débitos ≠ créditos")
	ErrVoucherTypeUnknown  = errors.New("tipo de comprobante no registrado — regístralo primero en Configuración de comprobantes")
	ErrVoucherTypeNotFound = errors.New("tipo de comprobante no encontrado")
	// ErrNumberCounterInvalid / ErrNumberCounterBackwards — al fijar manualmente el consecutivo de
	// un tipo de comprobante (ver application.GetJournalUseCase.SetVoucherCounter).
	ErrNumberCounterInvalid   = errors.New("el próximo número debe ser mayor o igual a 1")
	ErrNumberCounterBackwards = errors.New("el consecutivo indicado ya fue superado — no se puede retroceder")
)
