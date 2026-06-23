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

	// ErrIssuerAccessDenied: SelectIssuer pidió activar una empresa a la que userID no tiene
	// vínculo en user_issuers — mismo 404-en-espíritu que el resto de la API ("no existe" e
	// "ID válido pero no es tuyo" se tratan igual), aunque acá se distingue porque no hay un
	// recurso al que devolverle un 404 propiamente.
	ErrIssuerAccessDenied = errors.New("auth: no tienes acceso a esa empresa")
)
