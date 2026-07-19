// Package qr construye la URL del código QR que exige la representación gráfica de los
// documentos electrónicos DIAN (Anexo Técnico 1.9, sección 11.7.1).
package qr

import (
	"fmt"
	"strconv"

	"github.com/diegofxm/cofacture/domain"
)

const (
	habilitacionBaseURL = "https://catalogo-vpfe-hab.dian.gov.co/document/searchqr"
	produccionBaseURL   = "https://catalogo-vpfe.dian.gov.co/document/searchqr"
)

// URL construye la URL del QR a partir del CUFE (o CUDE) y el código de ambiente
// ("1" producción, "2" habilitación — el mismo valor que cbc:ProfileExecutionID). La DIAN
// usa dominios distintos por ambiente; confirmado contra dos facturas reales de producción
// y el texto del anexo técnico.
func URL(environmentCode, documentKey string) string {
	base := produccionBaseURL
	if environmentCode == "2" {
		base = habilitacionBaseURL
	}
	return base + "?documentkey=" + documentKey
}

// SupportDocumentURL construye la URL del QR del Documento Soporte.
// Usa el mismo endpoint searchqr que FE/NC/ND (verificado: FindDocument no redirige).
func SupportDocumentURL(environmentCode, cuds string) string {
	return URL(environmentCode, cuds)
}

// AdjustmentNoteContent construye el contenido completo del QR de la Nota de Ajuste al DS
// (InvoiceTypeCode "95"). Sigue el mismo patrón que SupportDocumentContent (bloque de texto
// multilinea seguido de la URL), adaptado para el tipo de documento NA.
func AdjustmentNoteContent(inv domain.Invoice, cuds, softwarePIN string) string {
	var codImp, valImp string
	for _, t := range inv.HeaderTaxes {
		if t.TypeCode == "01" {
			codImp = t.TypeCode
			valImp = domain.FormatCents(t.TaxAmountCents)
			break
		}
	}
	if codImp == "" && len(inv.HeaderTaxes) > 0 {
		codImp = inv.HeaderTaxes[0].TypeCode
		valImp = domain.FormatCents(inv.HeaderTaxes[0].TaxAmountCents)
	}
	if codImp == "" {
		codImp = "01"
		valImp = "0.00"
	}

	url := URL(inv.EnvironmentCode, cuds)

	ambLabel := strconv.Itoa(func() int {
		if inv.EnvironmentCode == "2" {
			return 2
		}
		return 1
	}())

	return fmt.Sprintf(
		"N°NotaAjuste=%s\nFecha=%s\nHora=%s\nValNA=%s\nCodImp=%s\nValImp=%s\nValTot=%s\nNumSNO=%s\nNITABS=%s\nPIN:%s\nAmb:%s\nCUDS=%s\nURL=%s",
		inv.Prefix+inv.Number,
		inv.IssueDate,
		inv.IssueTime,
		domain.FormatCents(inv.Totals.LineExtensionCents),
		codImp,
		valImp,
		domain.FormatCents(inv.Totals.PayableCents),
		inv.Supplier.Identification.Number,
		inv.Customer.Identification.Number,
		softwarePIN,
		ambLabel,
		cuds,
		url,
	)
}

// SupportDocumentContent construye el contenido completo del QR del Documento Soporte
// (InvoiceTypeCode "05"). A diferencia de la Factura/NC/ND cuyo QR es solo una URL, el DS
// exige un bloque de texto multilinea seguido de la URL (Anexo Técnico 1.9, sección 11.7.1).
//
// softwarePIN es el PIN del software autorizado — el mismo usado para calcular el CUDS.
// cuds es el CUDS ya calculado (hex SHA-384).
//
// Roles en DS: Supplier = tercero no obligado (NumSNO), Customer = empresa emisora (NITABS).
func SupportDocumentContent(inv domain.Invoice, cuds, softwarePIN string) string {
	var codImp, valImp string
	for _, t := range inv.HeaderTaxes {
		if t.TypeCode == "01" { // IVA
			codImp = t.TypeCode
			valImp = domain.FormatCents(t.TaxAmountCents)
			break
		}
	}
	if codImp == "" && len(inv.HeaderTaxes) > 0 {
		codImp = inv.HeaderTaxes[0].TypeCode
		valImp = domain.FormatCents(inv.HeaderTaxes[0].TaxAmountCents)
	}
	if codImp == "" {
		codImp = "01"
		valImp = "0.00"
	}

	url := URL(inv.EnvironmentCode, cuds)

	ambLabel := strconv.Itoa(func() int {
		if inv.EnvironmentCode == "2" {
			return 2
		}
		return 1
	}())

	return fmt.Sprintf(
		"N°DocSoporte=%s\nFecha=%s\nHora=%s\nValDS=%s\nCodImp=%s\nValImp=%s\nValTot=%s\nNumSNO=%s\nNITABS=%s\nPIN:%s\nAmb:%s\nCUDS=%s\nURL=%s",
		inv.Prefix+inv.Number,
		inv.IssueDate,
		inv.IssueTime,
		domain.FormatCents(inv.Totals.LineExtensionCents),
		codImp,
		valImp,
		domain.FormatCents(inv.Totals.PayableCents),
		inv.Supplier.Identification.Number,
		inv.Customer.Identification.Number,
		softwarePIN,
		ambLabel,
		cuds,
		url,
	)
}
