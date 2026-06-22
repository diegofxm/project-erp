package customers

import "errors"

var (
	ErrCustomerNotFound    = errors.New("customers: cliente no encontrado")
	ErrEmptyName           = errors.New("customers: el nombre es obligatorio")
	ErrEmptyIdentification = errors.New("customers: el número de identificación es obligatorio")
)
