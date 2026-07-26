package withholdings

import "errors"

var (
	ErrConceptNotFound = errors.New("concepto de retención no encontrado")
	ErrUVTNotFound     = errors.New("valor de UVT no registrado para el año solicitado")
)
