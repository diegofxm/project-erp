package auth

import "errors"

var (
	ErrUserNotFound       = errors.New("auth: usuario no encontrado")
	ErrEmailAlreadyExists = errors.New("auth: ya existe un usuario con ese correo")
	ErrEmptyEmail         = errors.New("auth: el correo es obligatorio")
	ErrEmptyPassword      = errors.New("auth: la contraseña es obligatoria")
	ErrPasswordTooShort   = errors.New("auth: la contraseña debe tener al menos 8 caracteres")
	ErrEmptyName          = errors.New("auth: el nombre es obligatorio")

	// ErrInvalidCredentials se devuelve tanto si el correo no existe como si la contraseña no
	// coincide — nunca se distingue cuál de los dos falló, para no confirmarle a un atacante
	// que un correo en particular sí está registrado.
	ErrInvalidCredentials = errors.New("auth: correo o contraseña incorrectos")
	ErrUserInactive       = errors.New("auth: el usuario está inactivo")
	ErrInvalidToken       = errors.New("auth: token inválido o expirado")
)
