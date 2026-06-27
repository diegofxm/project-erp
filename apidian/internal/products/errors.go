package products

import "errors"

var (
	ErrProductNotFound  = errors.New("products: producto no encontrado")
	ErrEmptyDescription = errors.New("products: la descripción es obligatoria")
	ErrEmptyUnitCode    = errors.New("products: la unidad de medida es obligatoria")
	ErrInvalidUnitPrice = errors.New("products: el precio unitario no puede ser negativo")
	// ErrInvalidTaxTypeCode: el cliente ya no manda tax_type_name (ver
	// docs/apidian-architecture.md) — el servicio lo deriva del catálogo a partir de
	// TaxTypeCode, y este es el error si ese código no existe en tax_types.
	ErrInvalidTaxTypeCode = errors.New("products: tipo de impuesto (tax_type_code) inválido")
)
