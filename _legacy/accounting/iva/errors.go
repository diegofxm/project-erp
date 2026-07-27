package iva

import "errors"

var (
	ErrDeclarationNotFound    = errors.New("declaración de IVA no encontrada")
	ErrDeclarationAlreadyFiled = errors.New("la declaración ya fue radicada")
	ErrDeclarationAlreadyPaid  = errors.New("la declaración ya tiene asiento de pago")
	ErrNothingToPay           = errors.New("no hay IVA a pagar en esta declaración")
	ErrBankAccountRequired    = errors.New("se requiere cuenta bancaria para el asiento de pago")
)
