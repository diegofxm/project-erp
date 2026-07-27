package accounts

import "errors"

var (
	ErrAccountNotFound  = errors.New("accounts: cuenta no encontrada")
	ErrAccountNotActive = errors.New("accounts: cuenta inactiva")
	// ErrAccountNotPostable: solo cuentas de nivel 4 (is_posting=true) aceptan movimientos.
	// Los niveles 1-3 son agrupadores — registrar en ellos viola la estructura del PUC.
	ErrAccountNotPostable = errors.New("accounts: la cuenta no es de movimiento (is_posting=false)")
)
