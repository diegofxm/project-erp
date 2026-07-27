package domain

import "errors"

var (
	ErrUserNotFound    = errors.New("usuario no encontrado")
	ErrEmailTaken      = errors.New("correo ya registrado")
	ErrInvalidPassword = errors.New("correo o contraseña incorrectos")
	ErrInvalidToken    = errors.New("token inválido o expirado")
	ErrUserInactive    = errors.New("cuenta inactiva")
	ErrNotAMember      = errors.New("usuario no es miembro de esta empresa")
)
