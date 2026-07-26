package banking

import "errors"

var (
	ErrBankAccountNotFound    = errors.New("banking: cuenta bancaria no encontrada")
	ErrStatementLineNotFound  = errors.New("banking: línea de extracto no encontrada")
	ErrAlreadyReconciled      = errors.New("banking: la línea ya está conciliada")
	ErrNotReconciled          = errors.New("banking: la línea no está conciliada")
	ErrJournalLineNotFound    = errors.New("banking: línea de asiento no encontrada")
	ErrDuplicateAccountNo     = errors.New("banking: ya existe una cuenta bancaria con ese número para la empresa")
)
