package contracts

import "errors"

var (
	ErrNotFound        = errors.New("contracts: contrato no encontrado")
	ErrAlreadyActive   = errors.New("contracts: el empleado ya tiene un contrato activo")
	ErrNotActive       = errors.New("contracts: el contrato no está activo")
)
