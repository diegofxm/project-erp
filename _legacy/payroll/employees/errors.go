package employees

import "errors"

var (
	ErrNotFound = errors.New("employees: empleado no encontrado")
	ErrExists   = errors.New("employees: ya existe un empleado con ese número de identificación en esta empresa")
)
