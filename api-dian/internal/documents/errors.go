package documents

import "errors"

var (
	ErrDocumentNotFound = errors.New("documents: documento no encontrado")

	ErrMissingIssuer         = errors.New("documents: el emisor es obligatorio")
	ErrMissingNumberingRange = errors.New("documents: el rango de numeración es obligatorio")
	ErrEmptyLines            = errors.New("documents: el documento debe tener al menos una línea")
	ErrMissingCustomer       = errors.New("documents: el adquiriente es obligatorio")

	// ErrMissingBillingReference: CreditNote/DebitNote deben referenciar el CUFE del
	// documento que corrigen — no se puede emitir una nota "al aire".
	ErrMissingBillingReference = errors.New("documents: la nota debe referenciar el CUFE del documento que corrige")

	// ErrWrongDocumentType indica que el rango de numeración no corresponde al tipo de
	// documento que se está intentando emitir (ej. usar un rango de Nota Crédito para una
	// Factura).
	ErrWrongDocumentType = errors.New("documents: el rango de numeración no corresponde a este tipo de documento")
)
