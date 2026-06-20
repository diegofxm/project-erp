// Package dianhash contiene la lógica de concatenación de campos que comparten el CUFE
// (paquete cufe) y el CUDE (paquete cude) — Anexo Técnico 1.9, secciones 11.2 y 11.4. Es
// "internal": solo existe para que esos dos paquetes no dupliquen la misma fórmula, no es
// una API pública del módulo.
package dianhash

import "github.com/diegofxm/ubl21dian/domain"

// Seed construye la cadena de entrada del hash SHA-384 común a CUFE y CUDE: mismos campos,
// mismo orden. Lo único que cambia entre uno y otro es lastSeedComponent (la clave técnica
// del rango de numeración para CUFE, el PIN del software para CUDE) — quien llama decide
// cuál de los dos es.
func Seed(doc domain.Invoice, lastSeedComponent string) string {
	var ivaCents, incCents, icaCents int64
	for _, t := range doc.HeaderTaxes {
		switch t.TypeCode {
		case "01":
			ivaCents += t.TaxAmountCents
		case "04":
			incCents += t.TaxAmountCents
		case "03":
			icaCents += t.TaxAmountCents
		}
	}

	return doc.Prefix + doc.Number +
		doc.IssueDate +
		doc.IssueTime +
		domain.FormatCents(doc.Totals.LineExtensionCents) +
		"01" + domain.FormatCents(ivaCents) +
		"04" + domain.FormatCents(incCents) +
		"03" + domain.FormatCents(icaCents) +
		domain.FormatCents(doc.Totals.PayableCents) +
		doc.Supplier.Identification.Number +
		doc.Customer.Identification.Number +
		lastSeedComponent +
		doc.EnvironmentCode
}
