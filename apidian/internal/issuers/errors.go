package issuers

import "errors"

var (
	ErrIssuerNotFound     = errors.New("issuers: emisor no encontrado")
	ErrNITAlreadyExists   = errors.New("issuers: ya existe un emisor con ese NIT")
	ErrEmptyNIT           = errors.New("issuers: el NIT es obligatorio")
	ErrEmptyBusinessName  = errors.New("issuers: la razón social es obligatoria")
	ErrEmptySoftwareID    = errors.New("issuers: el Software ID de la DIAN es obligatorio")
	ErrEmptySoftwarePIN   = errors.New("issuers: el PIN del software es obligatorio")
	ErrEmptyCertificate   = errors.New("issuers: el certificado es obligatorio")
	ErrInvalidCertificate = errors.New("issuers: el certificado no se pudo leer con la contraseña dada — verifica el archivo .p12 y la contraseña")
	ErrInvalidEnvironment = errors.New(`issuers: el ambiente debe ser "1" (producción) o "2" (habilitación)`)
)
