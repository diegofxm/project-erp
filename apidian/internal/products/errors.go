package products

import "errors"

var (
	ErrProductNotFound  = errors.New("products: producto no encontrado")
	ErrEmptyDescription = errors.New("products: la descripción es obligatoria")
	ErrEmptyUnitCode    = errors.New("products: la unidad de medida es obligatoria")
	ErrInvalidUnitPrice = errors.New("products: el precio unitario no puede ser negativo")
)
