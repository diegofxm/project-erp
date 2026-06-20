package domain

// AttachedPartyInfo es la versión reducida de un tercero que usa el AttachedDocument
// (SenderParty/ReceiverParty) — solo lleva los datos tributarios básicos, a diferencia de
// Party (que además tiene dirección, contacto y representación legal, usados en
// AccountingSupplierParty/AccountingCustomerParty dentro del documento envuelto).
type AttachedPartyInfo struct {
	Name           string
	Identification Identification
	TaxRegimeCode  string // listName de TaxLevelCode
	LiabilityCodes []string
	TaxSchemeCode  string
	TaxSchemeName  string
}

// ValidationResult es la respuesta de validación de la DIAN para el documento envuelto,
// embebida en cac:ParentDocumentLineReference. El anexo técnico exige al menos una
// ocurrencia (AE38, "1..N") — el AttachedDocument es un artefacto posterior a la
// validación, no algo que se construye antes de tenerla.
type ValidationResult struct {
	LineID string // consecutivo, normalmente "1"

	DocumentID        string // ID del documento referenciado (Prefijo+Número)
	DocumentCUFE      string // o CUDE
	DocumentHashType  string // "CUFE-SHA384" o "CUDE-SHA384"
	DocumentIssueDate string

	// ApplicationResponseXML es el ApplicationResponse completo que devolvió la DIAN,
	// tal cual, para el CDATA de cac:Attachment/cac:ExternalReference/cbc:Description.
	ApplicationResponseXML string

	ValidatorID          string // fijo: "Unidad Especial Dirección de Impuestos y Aduanas Nacionales"
	ValidationResultCode string // ej. "02"
	ValidationDate       string
	ValidationTime       string
}

// AttachedDocument es el contenedor electrónico que se entrega al adquiriente: envuelve el
// documento firmado (Invoice/CreditNote/DebitNote) junto con la respuesta de validación de
// la DIAN.
type AttachedDocument struct {
	EnvironmentCode string

	// ID es el "consecutivo propio del generador del documento" (AE04b) — NO es el CUFE
	// del documento envuelto, son cosas distintas aunque en la práctica algunos
	// proveedores las confunden.
	ID string

	IssueDate string // fecha de generación del contenedor (>= IssueDate del documento envuelto)
	IssueTime string

	// ParentDocumentID es el cbc:ID del documento envuelto (Prefijo+Número), no del
	// contenedor.
	ParentDocumentID string

	Sender   AttachedPartyInfo
	Receiver AttachedPartyInfo

	// AttachmentXML es el XML firmado del documento envuelto (Invoice/CreditNote/
	// DebitNote), tal cual, para el CDATA de cac:Attachment/cac:ExternalReference/
	// cbc:Description.
	AttachmentXML string

	ValidationResults []ValidationResult
}
