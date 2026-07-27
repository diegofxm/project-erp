package tax

import "errors"

var (
	ErrRateNotFound        = errors.New("tax: tasa de renta no encontrada para el año")
	ErrTariffNotFound      = errors.New("tax: tarifa ICA no encontrada para municipio/CIIU/año")
	ErrAlreadyFiled        = errors.New("tax: declaración ya radicada — no se puede modificar")
	ErrAlreadyPaid         = errors.New("tax: declaración ya pagada")
	ErrNothingToPay        = errors.New("tax: no hay valor a pagar (saldo a favor o cero)")
	ErrBankAccountRequired = errors.New("tax: se requiere cuenta bancaria para el asiento de pago")
)
