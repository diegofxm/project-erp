package numbering

import "errors"

var (
	ErrRangeNotFound  = errors.New("numbering: rango de numeración no encontrado")
	ErrRangeExhausted = errors.New("numbering: el rango de numeración ya entregó todos los números autorizados")

	ErrMissingIssuer       = errors.New("numbering: el emisor es obligatorio")
	ErrMissingDocumentType = errors.New("numbering: el tipo de documento DIAN es obligatorio")
	ErrEmptyPrefix         = errors.New("numbering: el prefijo es obligatorio")
	ErrInvalidRange        = errors.New("numbering: range_from debe ser mayor a 0 y, si hay range_to, range_from <= range_to")
	ErrInvalidEnvironment  = errors.New(`numbering: el ambiente debe ser "1" (producción) o "2" (habilitación)`)

	// ErrNextNumberOutOfRange: el next_number opcional (ver Service.RegisterRange) debe caer
	// dentro de [range_from, range_to] — fuera de ese intervalo no tiene sentido como "número
	// que la DIAN ya autorizó bajo esta resolución".
	ErrNextNumberOutOfRange = errors.New("numbering: next_number debe estar dentro del rango [range_from, range_to]")

)
