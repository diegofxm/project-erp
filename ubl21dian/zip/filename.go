package zip

import "fmt"

// DocumentKind identifica el tipo de documento para el nombre de archivo XML
// (Anexo Técnico 1.9, sección 6.5.7).
type DocumentKind string

const (
	KindInvoice             DocumentKind = "fv" // Factura de Venta
	KindCreditNote          DocumentKind = "nc" // Nota Crédito
	KindDebitNote           DocumentKind = "nd" // Nota Débito
	KindApplicationResponse DocumentKind = "ar"
	KindAttachedDocument    DocumentKind = "ad"
)

// SoftwarePropioCode es el código de Proveedor Tecnológico ("ppp") para quien factura con
// software propio en vez de operar como PT de terceros (sección 6.5.8, nota).
const SoftwarePropioCode = "000"

// DocumentFileName construye el nombre de archivo XML que exige la sección 6.5.7:
//
//	{kind}{NIT sin DV, 10 dígitos}{código PT, 3 dígitos}{año, 2 dígitos}{consecutivo, 8 hex}.xml
//
// nit debe venir sin dígito de verificación. ptCode es el código de 3 dígitos asignado por
// la DIAN al proveedor tecnológico (SoftwarePropioCode si es software propio). year son los
// últimos 2 dígitos del año calendario. consecutive es el consecutivo de archivos enviados
// del tipo correspondiente — el anexo exige reiniciarlo a 1 cada 1 de enero; llevar la
// cuenta es responsabilidad de quien orquesta el envío (estado persistente), no de este
// paquete, que solo formatea.
//
// Nota: el texto de la sección 6.5.7/6.5.8 describe el consecutivo como hexadecimal
// ("en el rango 00000001 <= FFFFFFFF"), pero el ejemplo ilustrativo que trae el propio
// anexo para la "décima primera factura" muestra "00000011" — que en hexadecimal sería la
// 17ª, no la 11ª. Es una inconsistencia del documento, no algo que se pueda resolver desde
// aquí. Esta función sigue el texto de la regla (hex) porque es lo normativo; si al enviar
// contra habilitación la DIAN espera decimal, hay que ajustar esto.
func DocumentFileName(kind DocumentKind, nit, ptCode string, year int, consecutive uint32) string {
	return fmt.Sprintf("%s%010s%s%02d%08X.xml", kind, nit, ptCode, year%100, consecutive)
}

// PackageFileName construye el nombre del ZIP que exige la sección 6.5.8:
//
//	z{NIT sin DV, 10 dígitos}{código PT, 3 dígitos}{año, 2 dígitos}{consecutivo, 8 hex}.zip
//
// El consecutivo aquí es el del paquete comprimido enviado, distinto del consecutivo de
// archivos XML individuales de DocumentFileName.
func PackageFileName(nit, ptCode string, year int, consecutive uint32) string {
	return fmt.Sprintf("z%010s%s%02d%08X.zip", nit, ptCode, year%100, consecutive)
}
