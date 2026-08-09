package domain

import "errors"

var (
	ErrDocumentNotFound         = errors.New("documento no encontrado")
	ErrDocumentNotDraft         = errors.New("el documento ya no está en borrador")
	ErrRangeNotFound            = errors.New("rango de numeración no encontrado")
	ErrRangeExhausted           = errors.New("rango de numeración agotado")
	ErrRangeInactive            = errors.New("rango de numeración inactivo")
	ErrMissingCompany           = errors.New("company_id requerido")
	ErrMissingNumberingRange    = errors.New("numbering_range_id requerido")
	ErrEmptyLines               = errors.New("el documento debe tener al menos una línea")
	ErrMissingCustomer          = errors.New("datos del receptor/tercero requeridos")
	ErrMissingPaymentMeans      = errors.New("al menos un medio de pago requerido")
	ErrMissingBillingReference  = errors.New("billing_reference requerido para NC/ND/NA")
	ErrMissingSupplier          = errors.New("datos del tercero (supplier) requeridos para DS/NA")
	ErrCompanyNotReadyToIssue   = errors.New("la empresa no tiene software/certificado configurados")
	ErrCompanyNotFound          = errors.New("empresa no encontrada")
	ErrRangeCompanyMismatch     = errors.New("el rango no pertenece a esta empresa")
	ErrWrongDocumentType        = errors.New("el tipo de documento del rango no coincide")
	ErrInvalidPaymentTerm       = errors.New("forma de pago inválida")
	ErrInvalidPaymentMethod     = errors.New("medio de pago inválido")
	ErrInvalidLiabilityCode     = errors.New("código de responsabilidad tributaria inválido")
	ErrInvalidOperationTypeCode = errors.New("operation_type_code debe ser '10' o '11'")
	ErrInvalidTaxTypeCode       = errors.New("tipo de impuesto inválido")
	ErrInvalidItemStandardCode  = errors.New("código de estándar de producto inválido")

	// ErrDianRejectedSync marca un error de SendBillSync donde la DIAN respondió explícitamente
	// (soap:Fault) — a diferencia de un error de transporte (timeout, conexión), aquí no hay
	// ambigüedad: la solicitud sí llegó y la DIAN la procesó/rechazó a nivel de protocolo, así
	// que es seguro liberar el consecutivo. El adaptador de cofacture envuelve el soap.Fault
	// original con este sentinel (fmt.Errorf("%w: ...", ErrDianRejectedSync)) para que
	// application/confirm.go pueda distinguir el caso sin importar el paquete soap directamente.
	ErrDianRejectedSync = errors.New("la DIAN rechazó explícitamente la solicitud (fault de protocolo)")
)
