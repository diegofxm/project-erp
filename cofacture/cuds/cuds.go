// Package cuds calcula el Código Único de Documento Soporte (CUDS) según el Anexo
// Técnico 1.9 de la DIAN, sección 11.5.
//
// La fórmula del CUDS difiere del CUFE/CUDE: en lugar de tres slots fijos de impuesto
// (IVA+"01", INC+"04", ICA+"03"), el CUDS usa un único par CodImp+ValImp tomado del
// primer elemento de HeaderTaxes. Este comportamiento fue verificado contra el ejemplo
// oficial de la Caja de Herramientas DS v1.1 (DocumentoSoporte-OperacionConResidente.xml,
// CUDS=c96a728f…23a) y coincide con la estructura del contenido del QR del DS.
package cuds

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/diegofxm/cofacture/domain"
)

// Compute calcula el CUDS de un Documento Soporte (InvoiceTypeCode "05").
//
// softwarePIN es el PIN del software autorizado por la DIAN — el mismo valor que aparece en
// el campo PIN del contenido del QR y en sts:QRCode.
func Compute(doc domain.Invoice, softwarePIN string) string {
	var taxCode string
	var taxCents int64
	if len(doc.HeaderTaxes) > 0 {
		taxCode = doc.HeaderTaxes[0].TypeCode
		taxCents = doc.HeaderTaxes[0].TaxAmountCents
	}
	seed := doc.Prefix + doc.Number +
		doc.IssueDate + doc.IssueTime +
		domain.FormatCents(doc.Totals.LineExtensionCents) +
		taxCode + domain.FormatCents(taxCents) +
		domain.FormatCents(doc.Totals.PayableCents) +
		doc.Supplier.Identification.Number +
		doc.Customer.Identification.Number +
		softwarePIN +
		doc.EnvironmentCode
	sum := sha512.Sum384([]byte(seed))
	return hex.EncodeToString(sum[:])
}
