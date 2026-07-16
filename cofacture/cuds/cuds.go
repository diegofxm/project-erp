// Package cuds calcula el Código Único de Documento Soporte (CUDS) según el Anexo
// Técnico 1.9 de la DIAN, sección 11.5.
//
// La fórmula es SHA-384 sobre la misma cadena que CUFE y CUDE (paquete dianhash), pero el
// último componente (lastSeedComponent) es el PIN del software, no la clave técnica del rango.
// En el Documento Soporte (InvoiceTypeCode "05") los roles están invertidos respecto a la
// Factura: Supplier es el tercero no obligado (quien vende al emisor) y Customer es el emisor
// mismo (la empresa que adquiere). El hash refleja esa inversión porque los campos
// Supplier.Identification.Number y Customer.Identification.Number invierten su posición
// semántica, aunque la fórmula de concatenación es la misma.
package cuds

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/internal/dianhash"
)

// Compute calcula el CUDS de un Documento Soporte (InvoiceTypeCode "05").
//
// softwarePIN es el PIN del software autorizado por la DIAN — el mismo valor que aparece en
// el campo PIN del contenido del QR y en sts:QRCode. No viaja en el XML de forma explícita
// (solo en el QR), pero sí entra en el hash.
func Compute(doc domain.Invoice, softwarePIN string) string {
	sum := sha512.Sum384([]byte(dianhash.Seed(doc, softwarePIN)))
	return hex.EncodeToString(sum[:])
}
