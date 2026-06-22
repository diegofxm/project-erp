// Package xml contiene las constantes de namespaces UBL 2.1 + extensiones DIAN
// compartidas por todos los builders.
package xml

const (
	NSInvoiceDefault = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	NSCreditNote     = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
	NSDebitNote      = "urn:oasis:names:specification:ubl:schema:xsd:DebitNote-2"
	NSCac            = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	NSCbc            = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"
	NSDs             = "http://www.w3.org/2000/09/xmldsig#"
	NSExt            = "urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2"
	NSSts            = "dian:gov:co:facturaelectronica:Structures-2-1"
	NSXades          = "http://uri.etsi.org/01903/v1.3.2#"
	NSXades141       = "http://uri.etsi.org/01903/v1.4.1#"
	NSXsi            = "http://www.w3.org/2001/XMLSchema-instance"

	SchemaLocationInvoice = NSInvoiceDefault + " http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-Invoice-2.1.xsd"

	// DianAuthorizationProviderID es el NIT fijo de la DIAN como proveedor de autorización.
	DianAuthorizationProviderID = "800197268"
	// DianSchemeAgencyID identifica a la DIAN como agencia de los esquemas (catálogos).
	DianSchemeAgencyID   = "195"
	DianSchemeAgencyName = "CO, DIAN (Dirección de Impuestos y Aduanas Nacionales)"

	// IdentificationTypeNIT es el código de catálogo que dispara el dígito de verificación.
	IdentificationTypeNIT = "31"
)
