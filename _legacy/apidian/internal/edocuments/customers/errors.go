package customers

import "errors"

var (
	ErrCustomerNotFound    = errors.New("customers: cliente no encontrado")
	ErrEmptyName           = errors.New("customers: el nombre es obligatorio")
	ErrEmptyIdentification = errors.New("customers: el número de identificación es obligatorio")
	// ErrInvalidLiabilityCode: liability_codes es TEXT[], sin FK posible contra cada elemento
	// (ver CatalogPort en ports.go) — mismo motivo que issuers.ErrInvalidLiabilityCode.
	ErrInvalidLiabilityCode = errors.New("customers: responsabilidad fiscal (liability_codes) inválida")
	// ErrInvalidTaxSchemeCode: el cliente ya no manda tax_scheme_name (ver
	// docs/apidian-architecture.md) — el servicio lo deriva del catálogo a partir de
	// TaxSchemeCode, y este es el error si ese código no existe en tax_types.
	ErrInvalidTaxSchemeCode = errors.New("customers: tipo de régimen tributario (tax_scheme_code) inválido")
	// ErrInvalidIdentificationNumber: identification.type_code "31" (NIT) exige un número
	// puramente numérico — el dígito de verificación se deriva de él (ver internal/nit), nunca
	// se acepta del cliente.
	ErrInvalidIdentificationNumber = errors.New("customers: número de identificación inválido para NIT")
)
