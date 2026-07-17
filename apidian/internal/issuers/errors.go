package issuers

import "errors"

var (
	ErrIssuerNotFound     = errors.New("issuers: emisor no encontrado")
	ErrNITAlreadyExists   = errors.New("issuers: ya existe un emisor con ese NIT")
	ErrEmptyNIT           = errors.New("issuers: el NIT es obligatorio")
	ErrEmptyBusinessName  = errors.New("issuers: la razón social es obligatoria")
	ErrEmptySoftwareID    = errors.New("issuers: el Software ID de la DIAN es obligatorio")
	ErrEmptySoftwarePIN   = errors.New("issuers: el PIN del software es obligatorio")
	ErrEmptyNeSoftwareID  = errors.New("issuers: el Software ID de Nómina Electrónica (NE) es obligatorio")
	ErrEmptyNeSoftwarePIN = errors.New("issuers: el PIN del software NE es obligatorio")
	ErrEmptyCertificate   = errors.New("issuers: el certificado es obligatorio")
	ErrInvalidCertificate = errors.New("issuers: el certificado no se pudo leer con la contraseña dada — verifica el archivo .p12 y la contraseña")
	ErrInvalidEnvironment = errors.New(`issuers: el ambiente debe ser "1" (producción) o "2" (habilitación)`)
	// ErrTooManyIndustryClassificationCodes: máximo 4 códigos CIIU (1 actividad principal +
	// hasta 3 secundarias) — límite basado en la estructura real del RUT (Actividad principal/
	// Actividad secundaria/Otras actividades 1 y 2), no en una regla del Anexo Técnico (CIIU es
	// catálogo DANE, no DIAN, así que no hay validación DIAN que lo exija — esto es solo para
	// no aceptar algo que ningún RUT real podría tener).
	ErrTooManyIndustryClassificationCodes = errors.New("issuers: máximo 4 códigos CIIU (1 actividad principal + hasta 3 secundarias)")
	// ErrInvalidLiabilityCode: liability_codes es TEXT[], sin FK posible contra cada elemento
	// (ver CatalogPort en ports.go) — antes de esto, un código inválido aquí no se detectaba
	// nunca en este servicio, solo al confirmar un documento con la DIAN rechazándolo.
	ErrInvalidLiabilityCode = errors.New("issuers: responsabilidad fiscal (liability_codes) inválida")
	// ErrInvalidTaxSchemeCode: el cliente ya no manda tax_scheme_name (ver
	// docs/apidian-architecture.md) — el servicio lo deriva del catálogo a partir de
	// TaxSchemeCode, y este es el error si ese código no existe en tax_types.
	ErrInvalidTaxSchemeCode = errors.New("issuers: tipo de régimen tributario (tax_scheme_code) inválido")
	// ErrEmptyLogo/ErrInvalidLogoContentType: para la representación gráfica en PDF (ver
	// docs/apidian-architecture.md sección 9.39) — mismo criterio que ErrEmptyCertificate, un
	// puntero no-nil con datos vacíos es casi siempre un error de quien llama, nunca una forma
	// válida de "borrar" el logo.
	ErrEmptyLogo              = errors.New("issuers: el logo es obligatorio si se manda este campo")
	ErrInvalidLogoContentType = errors.New(`issuers: logo_content_type debe ser "png", "jpg" o "jpeg"`)
	// ErrInvalidIdentificationNumber: identification_type_code "31" (NIT) exige un NIT
	// puramente numérico — el dígito de verificación se deriva de él (ver internal/nit), nunca
	// se acepta del cliente.
	ErrInvalidIdentificationNumber = errors.New("issuers: NIT inválido para calcular el dígito de verificación")
)
