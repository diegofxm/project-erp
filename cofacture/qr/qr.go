// Package qr construye la URL del código QR que exige la representación gráfica de los
// documentos electrónicos DIAN (Anexo Técnico 1.9, sección 11.7.1).
package qr

import (
	"fmt"
	"strconv"

	"github.com/diegofxm/cofacture/domain"
)

const (
	habilitacionBaseURL   = "https://catalogo-vpfe-hab.dian.gov.co/document/searchqr"
	produccionBaseURL     = "https://catalogo-vpfe.dian.gov.co/document/searchqr"
	habilitacionDSBaseURL = "https://catalogo-vpfe-hab.dian.gov.co/Document/FindDocument"
	produccionDSBaseURL   = "https://catalogo-vpfe.dian.gov.co/Document/FindDocument"
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

// SupportDocumentURL construye la URL que va en sts:QRCode del Documento Soporte.
// Para DS la DIAN usa el endpoint FindDocument (con documentKey en camelCase) en lugar de
// searchqr — distinto dominio y path que FE/NC/ND.
func SupportDocumentURL(environmentCode, cuds string) string {
	base := produccionDSBaseURL
	if environmentCode == "2" {
		base = habilitacionDSBaseURL
	}
	return base + "?documentKey=" + cuds
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

	base := produccionDSBaseURL
	if inv.EnvironmentCode == "2" {
		base = habilitacionDSBaseURL
	}
	url := base + "?documentKey=" + cuds

	ambLabel := strconv.Itoa(func() int {
		if inv.EnvironmentCode == "2" {
			return 2
		}
		return 1
	}())

	return fmt.Sprintf(
		"N°DocSoporte=%s\nFecha=%s\nHora=%s\nValDS=%s\nCodImp=%s\nValImp=%s\nValTot=%s\nNumSNO=%s\nNITABS=%s\nPIN=%s\nAmb=%s\nCUDS=%s\n%s",
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
