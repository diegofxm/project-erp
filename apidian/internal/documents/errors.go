package documents

import "errors"

var (
	ErrDocumentNotFound = errors.New("documents: documento no encontrado")

	ErrMissingIssuer         = errors.New("documents: el emisor es obligatorio")
	ErrMissingNumberingRange = errors.New("documents: el rango de numeración es obligatorio")
	ErrEmptyLines            = errors.New("documents: el documento debe tener al menos una línea")
	ErrMissingCustomer       = errors.New("documents: el adquiriente es obligatorio")

	// ErrMissingPaymentMeans: cac:PaymentMeans es obligatorio para Invoice/CreditNote/
	// DebitNote según el Anexo Técnico (cardinalidad 1..N, ver docs/reference/
	// anexo-tecnico-1.9.txt FAN01/CAN01/DAN01) — la DIAN rechaza con "errores en campos
	// mandatorios" si no se informa ninguno. Se valida desde el borrador, no solo al
	// confirmar, para no gastar un número real en un documento que de todas formas va a
	// rechazar (ver docs/apidian-architecture.md sección 9.30).
	ErrMissingPaymentMeans = errors.New("documents: el documento debe tener al menos una forma de pago (payment_means)")

	// ErrInvalidPaymentTerm/ErrInvalidPaymentMethod: payment_means vive en JSONB sin FK posible
	// (ver CatalogPort en ports.go) — sin esto, un código inválido solo se detectaba al
	// confirmar, con la DIAN rechazando el documento ya con un número real reclamado.
	ErrInvalidPaymentTerm   = errors.New("documents: forma de pago (payment_means[].code) inválida")
	ErrInvalidPaymentMethod = errors.New("documents: medio de pago (payment_means[].payment_method_code) inválido")

	// ErrInvalidLiabilityCode: customer.liability_codes es TEXT[] pass-through del request,
	// sin FK posible contra cada elemento (ver CatalogPort en ports.go) — mismo motivo y mismo
	// momento de validación que ErrInvalidPaymentTerm/ErrInvalidPaymentMethod.
	ErrInvalidLiabilityCode = errors.New("documents: responsabilidad fiscal (customer.liability_codes) inválida")

	// ErrMissingBillingReference: CreditNote/DebitNote deben referenciar el CUFE del
	// documento que corrigen — no se puede emitir una nota "al aire".
	ErrMissingBillingReference = errors.New("documents: la nota debe referenciar el CUFE del documento que corrige")

	// ErrWrongDocumentType indica que el rango de numeración no corresponde al tipo de
	// documento que se está intentando emitir (ej. usar un rango de Nota Crédito para una
	// Factura).
	ErrWrongDocumentType = errors.New("documents: el rango de numeración no corresponde a este tipo de documento")

	// ErrNumberingRangeIssuerMismatch indica que el rango de numeración no pertenece al
	// emisor con el que se intenta emitir — sin este chequeo, sería posible reclamar
	// consecutivos de la resolución DIAN de OTRO emisor. Con internal/auth en juego, además
	// cierra la puerta a que un usuario emita documentos usando el rango de otro tenant.
	ErrNumberingRangeIssuerMismatch = errors.New("documents: el rango de numeración no pertenece a este emisor")

	// ErrCustomerIssuerMismatch indica que el CustomerID opcional (trazabilidad, ver model.go)
	// no pertenece al emisor con el que se intenta emitir — mismo criterio que
	// ErrNumberingRangeIssuerMismatch: nunca confiar en que un ID referenciado por el cliente
	// realmente le pertenece sin comprobarlo.
	ErrCustomerIssuerMismatch = errors.New("documents: el cliente referenciado no pertenece a este emisor")

	// ErrDocumentNotDraft: Update/Delete solo aplican mientras Status == StatusDraft — una vez
	// confirmado (número reclamado, firmado, posiblemente enviado), el documento es inmutable
	// para siempre, mismo principio que "nunca mutar un documento ya firmado" (ver model.go).
	ErrDocumentNotDraft = errors.New("documents: el documento ya fue confirmado, no se puede editar ni eliminar")

	// ErrIssuerNotReadyToIssue: el emisor todavía no tiene software_id/software_pin/
	// certificado configurados (ver internal/issuers) — no se puede confirmar (firmar) un
	// documento sin esos datos. Mensaje propio, más claro que el error de bajo nivel que
	// lanzaría signer.LoadPKCS12 al recibir un certificado vacío.
	ErrIssuerNotReadyToIssue = errors.New("documents: el emisor todavía no tiene software/certificado configurados — complétalos con PUT /issuers/me antes de confirmar")

	// ErrInvalidTaxSchemeCode/ErrInvalidTaxTypeCode: el cliente ya no manda customer.
	// tax_scheme_name ni lines[].tax_type_name (ver docs/apidian-architecture.md) — el
	// servicio los deriva del catálogo a partir de sus códigos, y estos son los errores si
	// esos códigos no existen en tax_types.
	ErrInvalidTaxSchemeCode = errors.New("documents: tipo de régimen tributario del cliente (customer.tax_scheme_code) inválido")
	ErrInvalidTaxTypeCode   = errors.New("documents: tipo de impuesto de línea (lines[].tax_type_code) inválido")

	// ErrInvalidItemStandardCode: lines[].item_type_code ya no es un campo libre — la DIAN
	// rechaza el documento si no coincide con la tabla 13.3.5 del Anexo Técnico (ver
	// docs/apidian-architecture.md sección 9.45). Igual criterio que ErrInvalidTaxTypeCode.
	ErrInvalidItemStandardCode = errors.New("documents: estándar de clasificación de línea (lines[].item_type_code) inválido")

	// ErrPDFNotSupportedForDocumentType: la representación gráfica (ver
	// docs/apidian-architecture.md sección 9.39) por ahora solo está implementada para
	// Factura — Nota Crédito/Nota Débito se agregan cuando el ciclo de Factura esté probado
	// de punta a punta (PDF + correo).
	ErrPDFNotSupportedForDocumentType = errors.New("documents: la representación gráfica todavía no está disponible para este tipo de documento")

	// ErrEmailNotSupportedForDocumentType: mismo criterio que ErrPDFNotSupportedForDocumentType
	// — el envío por correo (ver docs/apidian-architecture.md sección 9.42) por ahora solo
	// está implementado para Factura.
	ErrEmailNotSupportedForDocumentType = errors.New("documents: el envío por correo todavía no está disponible para este tipo de documento")

	// ErrDocumentNotAccepted: solo se envía al cliente un documento que la DIAN ya aceptó
	// (StatusAccepted) — nunca un borrador, uno rechazado/con error de envío, ni uno en
	// StatusSent (resultado todavía desconocido, podría acabar rechazado).
	ErrDocumentNotAccepted = errors.New("documents: el documento debe estar aceptado por la DIAN antes de enviarlo al cliente")

	// ErrCustomerEmailMissing: el snapshot del cliente (ver model.go) no trae correo — sin él
	// no hay a dónde enviar la factura.
	ErrCustomerEmailMissing = errors.New("documents: el cliente no tiene un correo registrado, no se puede enviar la factura")
)
