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

	// ErrInvalidItemStandardCode: item_type_code ya no es un campo libre — debe coincidir con
	// la tabla 13.3.5 del Anexo Técnico (ver docs/apidian-architecture.md sección 9.45), igual
	// criterio que ErrInvalidTaxTypeCode.
	ErrInvalidItemStandardCode = errors.New("products: estándar de clasificación (item_type_code) inválido")

	// ErrDuplicateItemCode: item_code es único por emisor cuando no está vacío (índice parcial,
	// ver migración 000011) — dos productos del mismo emisor no pueden compartir el mismo
	// código de ítem.
	ErrDuplicateItemCode = errors.New("products: ya existe un producto con ese código de ítem")
)
