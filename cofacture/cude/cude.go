// Package cude calcula el Código Único de Documento Electrónico (CUDE) de Notas Crédito y
// Notas Débito, según el Anexo Técnico 1.9 de la DIAN (secciones 11.4.3-11.4.6).
//
// El CUFE (Factura) vive en el paquete cufe, no aquí — son fórmulas casi idénticas, pero
// identificadores distintos para documentos distintos.
package cude

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/internal/dianhash"
)

// Compute calcula el CUDE de una Nota Crédito o Nota Débito.
//
// La fórmula es idéntica a la del CUFE en estructura y orden de campos (validado contra los
// dos ejemplos oficiales del anexo, uno por cada tipo de nota) — la única diferencia real es
// que en vez de la clave técnica del rango de numeración se usa el PIN del software
// ("Software-PIN", el mismo que se usa para securitycode.Compute). noteBase es el Invoice
// embebido en domain.CreditNote/domain.DebitNote (campo promovido, ej. creditNote.Invoice).
func Compute(noteBase domain.Invoice, softwarePIN string) string {
	sum := sha512.Sum384([]byte(dianhash.Seed(noteBase, softwarePIN)))
	return hex.EncodeToString(sum[:])
}
