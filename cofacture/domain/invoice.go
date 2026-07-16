package domain

// Invoice es el modelo de una Factura Electrónica de Venta (InvoiceTypeCode "01").
//
// CUFE, SoftwareSecurityCode y QRURL se dejan vacíos al construir el documento: se calculan
// en pasos posteriores del pipeline (cufe, qr) a partir de estos mismos campos y se inyectan
// en el XML ya construido antes de firmar.
type Invoice struct {
	ProfileID         string // "DIAN 2.1"
	EnvironmentCode   string // "1" producción, "2" pruebas (también usado como schemeID de UUID)
	OperationTypeCode string // catálogo de tipos de operación, ej. "10" = Estándar
	DocumentTypeCode  string // "01" factura de venta nacional
	HashType          string // "CUFE-SHA384", usado como schemeName de UUID

	Prefix string
	Number string

	IssueDate string // YYYY-MM-DD
	IssueTime string // HH:MM:SS-05:00
	DueDate   string // opcional

	Note string // opcional

	CurrencyCode string // catálogo currency_codes, ISO 4217, ej. "COP"

	OrderReferenceNumber string // opcional

	Supplier Party
	Customer Party

	PaymentMeans []PaymentMean

	// HeaderTaxes son los TaxTotal de cabecera (uno por tipo de impuesto, agregando todas
	// las líneas). El cálculo de estos totales es responsabilidad de quien construye el
	// modelo, no del builder.
	HeaderTaxes []Tax

	// WithholdingTaxes son las retenciones (WithholdingTaxTotal) — exclusivo del Documento
	// Soporte (InvoiceTypeCode "05"). Cada elemento genera un cac:WithholdingTaxTotal
	// independiente (uno por tipo: ReteIVA="05", ReteRenta="06"). Se ignora en Invoice/NC/ND.
	WithholdingTaxes []Tax

	Totals Totals
	Lines  []Line

	NumberingRange   NumberingRange
	SoftwareProvider SoftwareProvider

	CUFE                 string
	SoftwareSecurityCode string
	QRURL                string
}
