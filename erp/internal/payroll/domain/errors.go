package domain

import "errors"

var (
	ErrEmployeeNotFound    = errors.New("payroll: empleado no encontrado")
	ErrEmployeeExists      = errors.New("payroll: empleado ya existe con ese documento")
	ErrContractNotFound    = errors.New("payroll: contrato no encontrado")
	ErrNoActiveContract    = errors.New("payroll: el empleado no tiene contrato activo")
	ErrPayslipNotFound     = errors.New("payroll: liquidación no encontrada")
	ErrPayslipExists       = errors.New("payroll: ya existe una liquidación para ese período")
	ErrPayslipNotDraft     = errors.New("payroll: la liquidación no está en estado borrador")
	ErrPayslipNotApproved  = errors.New("payroll: la liquidación no está aprobada")
	ErrNoSMMLV             = errors.New("payroll: no hay SMMLV registrado para el año")
	ErrNoARLRate           = errors.New("payroll: no hay tasa ARL registrada para el año y clase de riesgo")
)
