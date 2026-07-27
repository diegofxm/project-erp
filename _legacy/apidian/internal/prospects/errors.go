package prospects

import "errors"

var (
	ErrNotFound      = errors.New("solicitud no encontrada")
	ErrDuplicateEmail = errors.New("ya existe una solicitud con ese correo")
	ErrAlreadyReviewed = errors.New("la solicitud ya fue procesada")
)
