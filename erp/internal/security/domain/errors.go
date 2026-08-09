package domain

import "errors"

var (
	ErrUserNotFound    = errors.New("usuario no encontrado")
	ErrEmailTaken      = errors.New("correo ya registrado")
	ErrInvalidPassword = errors.New("correo o contraseña incorrectos")
	ErrInvalidToken    = errors.New("token inválido o expirado")
	ErrUserInactive    = errors.New("cuenta inactiva")
	ErrNotAMember      = errors.New("usuario no es miembro de esta empresa")
	// ErrCurrentPasswordIncorrect — distinta de ErrInvalidPassword (esa habla de "correo o
	// contraseña", pensada para el login; aquí no hay correo de por medio, solo la contraseña
	// actual que el usuario ya autenticado escribió mal al querer cambiarla).
	ErrCurrentPasswordIncorrect = errors.New("la contraseña actual no es correcta")
	// ErrTooManyAttempts: el LoginRateLimiter bloqueó el correo por demasiados intentos fallidos
	// recientes -- ver LoginUseCase.
	ErrTooManyAttempts = errors.New("demasiados intentos fallidos, intenta de nuevo más tarde")
)
