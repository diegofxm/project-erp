package builder

import (
	"strconv"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
	ubl "github.com/diegofxm/cofacture/xml"
)

// BuildCreditNote construye el árbol XML de una Nota Crédito. Reutiliza la misma
// infraestructura que Invoice (extensiones DIAN, partes, impuestos, totales, líneas) — la
// estructura es casi idéntica, lo que cambia es el elemento raíz, el tipo de documento y la
// referencia obligatoria a la factura corregida.
func BuildCreditNote(cn domain.CreditNote) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="no"`)

	root := doc.CreateElement("CreditNote")
	root.CreateAttr("xmlns", ubl.NSCreditNote)
	root.CreateAttr("xmlns:cac", ubl.NSCac)
	root.CreateAttr("xmlns:cbc", ubl.NSCbc)
	root.CreateAttr("xmlns:ds", ubl.NSDs)
	root.CreateAttr("xmlns:ext", ubl.NSExt)
	root.CreateAttr("xmlns:sts", ubl.NSSts)
	root.CreateAttr("xmlns:xades", ubl.NSXades)
	root.CreateAttr("xmlns:xades141", ubl.NSXades141)
	root.CreateAttr("xmlns:xsi", ubl.NSXsi)
	root.CreateAttr("xsi:schemaLocation", ubl.NSCreditNote+" http://docs.oasis-open.org/ubl/os-UBL-2.1/xsd/maindoc/UBL-CreditNote-2.1.xsd")

	appendUBLExtensions(root, cn.Invoice)

	root.CreateElement("cbc:UBLVersionID").SetText("UBL 2.1")
	root.CreateElement("cbc:CustomizationID").SetText(cn.OperationTypeCode)
	root.CreateElement("cbc:ProfileID").SetText(cn.ProfileID)
	root.CreateElement("cbc:ProfileExecutionID").SetText(cn.EnvironmentCode)
	root.CreateElement("cbc:ID").SetText(cn.Prefix + cn.Number)

	uuid := root.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeID", cn.EnvironmentCode)
	uuid.CreateAttr("schemeName", cn.HashType)
	uuid.SetText(cn.CUFE) // el CUDE de esta nota, ver nota en domain.CreditNote

	root.CreateElement("cbc:IssueDate").SetText(cn.IssueDate)
	root.CreateElement("cbc:IssueTime").SetText(cn.IssueTime)
	root.CreateElement("cbc:CreditNoteTypeCode").SetText(cn.CreditNoteTypeCode)
	if cn.Note != "" {
		root.CreateElement("cbc:Note").SetText(cn.Note)
	}

	currency := root.CreateElement("cbc:DocumentCurrencyCode")
	currency.CreateAttr("listAgencyID", "6")
	currency.CreateAttr("listAgencyName", "United Nations Economic Commission for Europe")
	currency.CreateAttr("listID", "ISO 4217 Alpha")
	currency.SetText(cn.CurrencyCode)

	root.CreateElement("cbc:LineCountNumeric").SetText(strconv.Itoa(len(cn.Lines)))

	if cn.DiscrepancyResponse != nil {
		appendDiscrepancyResponse(root, *cn.DiscrepancyResponse)
	}
	appendBillingReference(root, "InvoiceDocumentReference", cn.BillingReference)

	appendAccountingParty(root, "AccountingSupplierParty", cn.Supplier, true, cn.Prefix, true)
	appendAccountingParty(root, "AccountingCustomerParty", cn.Customer, false, "", false)

	for _, pm := range cn.PaymentMeans {
		appendPaymentMean(root, pm)
	}

	appendTaxTotal(root, cn.HeaderTaxes, cn.CurrencyCode)
	appendMonetaryTotal(root, "LegalMonetaryTotal", cn.Totals, cn.CurrencyCode, cn.DocumentTypeCode)

	for i, line := range cn.Lines {
		appendDocumentLine(root, "CreditNoteLine", "CreditedQuantity", i+1, line, cn.CurrencyCode, cn.DocumentTypeCode, cn.Supplier.Identification, "")
	}

	return doc, nil
}
