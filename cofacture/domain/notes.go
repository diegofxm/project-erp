package domain

// BillingReference identifica la factura que una Nota Crédito/Débito corrige
// (cac:BillingReference) — siempre obligatoria en ambas, a diferencia de
// DiscrepancyResponse.
type BillingReference struct {
	Prefix    string
	Number    string
	CUFE      string // CUFE de la factura referenciada (no el de esta nota)
	IssueDate string
}

// DiscrepancyResponse es el motivo de la nota (cac:DiscrepancyResponse) — obligatorio
// cuando la nota referencia una factura específica (operación "20" para Nota Crédito, "30"
// para Nota Débito, catálogo de tipos de operación). ResponseCode usa el catálogo de
// conceptos de nota crédito/débito.
type DiscrepancyResponse struct {
	ReferenceID  string // normalmente Prefix+Number de la factura referenciada
	ResponseCode string
	Description  string
}

// CreditNote es una Nota Crédito. Comparte casi todos los campos con Invoice — UBL las
// trata como documentos distintos, pero lo que de verdad cambia es el tipo de documento, la
// referencia a la factura corregida y el motivo. Por eso se reutiliza Invoice en vez de
// repetir ~20 campos.
//
// El campo heredado CUFE guarda en realidad el CUDE de esta nota (mismo slot que usa la
// DIAN — cbc:UUID — para cualquiera de los documentos electrónicos; HashType es lo que
// distingue "CUFE-SHA384" de "CUDE-SHA384", no el nombre del campo en este modelo).
type CreditNote struct {
	Invoice

	CreditNoteTypeCode  string // catálogo de tipo de Nota Crédito (cbc:CreditNoteTypeCode)
	BillingReference    BillingReference
	DiscrepancyResponse *DiscrepancyResponse // nil si la operación no exige motivo
}

// DebitNote es una Nota Débito. Estructuralmente igual a CreditNote — la única diferencia
// de fondo con el builder es que usa cac:RequestedMonetaryTotal en vez de
// cac:LegalMonetaryTotal, y el anexo técnico no define un equivalente a
// cbc:CreditNoteTypeCode para Nota Débito (verificado: no hay tal elemento entre
// DocumentCurrencyCode y LineCountNumeric en la sección 8.4 del anexo).
type DebitNote struct {
	Invoice

	BillingReference    BillingReference
	DiscrepancyResponse *DiscrepancyResponse
}
