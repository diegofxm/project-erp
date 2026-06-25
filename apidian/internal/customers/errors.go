package customers

import "errors"

var (
	ErrCustomerNotFound    = errors.New("customers: cliente no encontrado")
	ErrEmptyName           = errors.New("customers: el nombre es obligatorio")
	ErrEmptyIdentification = errors.New("customers: el número de identificación es obligatorio")
	// ErrInvalidLiabilityCode: liability_codes es TEXT[], sin FK posible contra cada elemento
	// (ver CatalogPort en ports.go) — mismo motivo que issuers.ErrInvalidLiabilityCode.
	ErrInvalidLiabilityCode = errors.New("customers: responsabilidad fiscal (liability_codes) inválida")
)
