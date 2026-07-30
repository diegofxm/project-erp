// Package domain contiene los tipos de dominio que alimentan los builders de cofacture.
// Son structs planos sin persistencia (sin tags db/gorm, sin SQL) — el equivalente a un DTO.
//
// Estos tipos no validan códigos de catálogo DIAN (tipo de documento, impuesto, medio de
// pago, etc.) — reciben los códigos ya validados por quien los construya (el orquestador).
// El core solo sabe en qué nodo del XML va cada código, no si el código es válido.
package domain

// Identification representa el número de identificación de un tercero (NIT, CC, etc.)
// junto con los atributos que la DIAN exige en los esquemas schemeID/schemeName.
type Identification struct {
	Number           string // ej. "900123456"
	TypeCode         string // catálogo identification_types, ej. "31" = NIT
	VerificationCode string // dígito de verificación, solo aplica cuando TypeCode == "31"
}

// Address es la dirección física/de registro de un Party.
type Address struct {
	Line        string
	CityCode    string
	CityName    string
	PostalZone  string
	StateCode   string
	StateName   string
	CountryCode string
	CountryName string
}

// Party es un emisor o receptor (AccountingSupplierParty / AccountingCustomerParty).
type Party struct {
	EntityTypeCode string // "1" persona natural, "2" persona jurídica
	Identification Identification
	Name           string
	Address        Address // se omite del XML si Line está vacío
	TaxRegimeCode  string  // catálogo type_regimes, va como listName de TaxLevelCode
	// LiabilityCodes son las responsabilidades tributarias (catálogo tax_level_codes,
	// ej. O-13, O-15, O-47). La DIAN permite más de una por tercero — se serializan
	// concatenadas con ";" en un único cbc:TaxLevelCode, no un elemento por código.
	LiabilityCodes []string
	// IndustryClassificationCodes son los códigos CIIU (catálogo DANE, no DIAN — por eso no
	// hay tabla de catálogo que los valide, solo un límite de cardinalidad que aplica quien
	// construya el Party, ver companies.Service.validateCompany). Solo aplica al emisor, nunca
	// al receptor (el Anexo Técnico lo describe como "código de actividad económica del
	// emisor"). Se serializan concatenados con ";" en un único cbc:IndustryClassificationCode.
	IndustryClassificationCodes []string
	TaxSchemeCode               string
	TaxSchemeName               string
	Phone                       string
	Email                       string
	// MerchantRegistrationNumber es el número de matrícula mercantil (Cámara de Comercio).
	// nil si no aplica (siempre nil para el receptor; opcional para el emisor — ej. una
	// persona natural sin registro mercantil no lo trae). El cbc:ID de
	// CorporateRegistrationScheme no es este número: es el prefijo de la factura.
	MerchantRegistrationNumber *string
}

// Tax es un subtotal de impuesto, a nivel de línea o de cabecera.
type Tax struct {
	TaxableAmountCents int64
	TaxAmountCents     int64
	Percent            float64
	TypeCode           string // catálogo tax_types, ej. "01" = IVA
	TypeName           string
}

// ReferencePrice se usa en muestras comerciales o ítems sin cargo (PricingReference).
type ReferencePrice struct {
	PriceAmountCents int64
	TypeCode         string
}

// Line es un renglón de detalle (InvoiceLine / CreditNoteLine / DebitNoteLine).
type Line struct {
	Description        string
	Quantity           float64
	UnitCode           string // catálogo unit_codes, ej. "94"
	LineExtensionCents int64
	UnitPriceCents     int64
	FreeOfCharge       bool
	ReferencePrice     *ReferencePrice // requerido cuando FreeOfCharge == true
	ItemCode           string
	ItemTypeCode       string // catálogo de estandares de codificación de producto (ej. "999")
	ItemTypeName       string
	ItemTypeAgencyID   string
	Taxes              []Tax
}

// PaymentMean es un medio de pago (PaymentMeans).
type PaymentMean struct {
	Code              string // forma de pago: contado/crédito (catálogo "payment_terms" del orquestador)
	PaymentMethodCode string // medio de pago propiamente: efectivo, transferencia... (catálogo "payment_methods" del orquestador)
	DueDate           string // solo aplica cuando Code == "2" (a crédito)
	PaymentReference  string
}

// Totals son los totales legales del documento (LegalMonetaryTotal).
type Totals struct {
	LineExtensionCents int64
	TaxExclusiveCents  int64
	TaxInclusiveCents  int64
	PrepaidCents       int64 // solo se serializa cuando > 0 y el documento es Invoice
	PayableCents       int64
}

// NumberingRange es la resolución de numeración autorizada por la DIAN para el rango
// que cubre este documento (InvoiceControl).
type NumberingRange struct {
	AuthorizedCode string
	Prefix         string
	StartNumber    string
	EndNumber      string
	StartDate      string // YYYY-MM-DD
	EndDate        string // YYYY-MM-DD
}

// SoftwareProvider identifica al proveedor tecnológico y al software autorizado por la DIAN.
type SoftwareProvider struct {
	ProviderIdentification Identification
	SoftwareID             string
}
