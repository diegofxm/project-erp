package vendors

import "errors"

var (
	ErrVendorNotFound              = errors.New("vendors: proveedor no encontrado")
	ErrEmptyName                   = errors.New("vendors: el nombre es obligatorio")
	ErrEmptyIdentification         = errors.New("vendors: el número de identificación es obligatorio")
	ErrInvalidLiabilityCode        = errors.New("vendors: responsabilidad fiscal (liability_codes) inválida")
	ErrInvalidTaxSchemeCode        = errors.New("vendors: tipo de régimen tributario (tax_scheme_code) inválido")
	ErrInvalidIdentificationNumber = errors.New("vendors: número de identificación inválido para NIT")
)
