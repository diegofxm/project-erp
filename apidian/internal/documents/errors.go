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
)
