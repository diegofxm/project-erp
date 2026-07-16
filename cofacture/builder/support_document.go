package builder

import (
	"strconv"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
	ubl "github.com/diegofxm/cofacture/xml"
)

// BuildSupportDocument construye el árbol XML de un Documento Soporte en adquisiciones
// efectuadas a no obligados a facturar (InvoiceTypeCode "05", CUDS-SHA384).
//
// La estructura es Invoice (mismo elemento raíz y namespaces que BuildInvoice) con tres
// diferencias clave:
//  1. Roles invertidos: Supplier = tercero no obligado (quien vende al emisor),
//     Customer = la empresa emisora (quien adquiere y genera el documento).
//  2. WithholdingTaxTotal en lugar de (o además de) TaxTotal para retenciones.
//  3. InvoiceControl presente (el DS tiene resolución DIAN propia, igual que la Factura).
//
// El llamador es responsable de calcular CUDS (paquete cuds), SoftwareSecurityCode y QRURL
// (paquete qr) antes de llamar esta función — los campos de inv se serializan tal cual.
func BuildSupportDocument(inv domain.Invoice) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="no"`)

	root := doc.CreateElement("Invoice")
	root.CreateAttr("xmlns", ubl.NSInvoiceDefault)
	root.CreateAttr("xmlns:cac", ubl.NSCac)
	root.CreateAttr("xmlns:cbc", ubl.NSCbc)
	root.CreateAttr("xmlns:ds", ubl.NSDs)
	root.CreateAttr("xmlns:ext", ubl.NSExt)
	root.CreateAttr("xmlns:sts", ubl.NSSts)
	root.CreateAttr("xmlns:xades", ubl.NSXades)
	root.CreateAttr("xmlns:xades141", ubl.NSXades141)
	root.CreateAttr("xmlns:xsi", ubl.NSXsi)
	root.CreateAttr("xsi:schemaLocation", ubl.SchemaLocationInvoice)

	// appendUBLExtensions maneja InvoiceControl para DocumentTypeCode "05"
	appendUBLExtensions(root, inv)

	root.CreateElement("cbc:UBLVersionID").SetText("UBL 2.1")
	root.CreateElement("cbc:CustomizationID").SetText(inv.OperationTypeCode) // "10" o "11"
	root.CreateElement("cbc:ProfileID").SetText(inv.ProfileID)
	root.CreateElement("cbc:ProfileExecutionID").SetText(inv.EnvironmentCode)
	root.CreateElement("cbc:ID").SetText(inv.Prefix + inv.Number)

	uuid := root.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeID", inv.EnvironmentCode)
	uuid.CreateAttr("schemeName", inv.HashType) // "CUDS-SHA384"
	uuid.SetText(inv.CUFE)                      // almacenamos el CUDS en el campo CUFE del modelo

	root.CreateElement("cbc:IssueDate").SetText(inv.IssueDate)
	root.CreateElement("cbc:IssueTime").SetText(inv.IssueTime)
	if inv.DueDate != "" {
		root.CreateElement("cbc:DueDate").SetText(inv.DueDate)
	}
	root.CreateElement("cbc:InvoiceTypeCode").SetText(inv.DocumentTypeCode) // "05"
	if inv.Note != "" {
		root.CreateElement("cbc:Note").SetText(inv.Note)
	}

	currency := root.CreateElement("cbc:DocumentCurrencyCode")
	currency.CreateAttr("listAgencyID", "6")
	currency.CreateAttr("listAgencyName", "United Nations Economic Commission for Europe")
	currency.CreateAttr("listID", "ISO 4217 Alpha")
	currency.SetText(inv.CurrencyCode)

	root.CreateElement("cbc:LineCountNumeric").SetText(strconv.Itoa(len(inv.Lines)))

	// Roles invertidos: Supplier = no-obligado, Customer = empresa compradora/emisora
	appendAccountingParty(root, "AccountingSupplierParty", inv.Supplier, false, "", false)
	appendAccountingParty(root, "AccountingCustomerParty", inv.Customer, true, inv.Prefix, true)

	for _, pm := range inv.PaymentMeans {
		appendPaymentMean(root, pm)
	}

	appendTaxTotal(root, inv.HeaderTaxes, inv.CurrencyCode)
	appendWithholdingTaxTotal(root, inv.WithholdingTaxes, inv.CurrencyCode)
	appendMonetaryTotal(root, "LegalMonetaryTotal", inv.Totals, inv.CurrencyCode, inv.DocumentTypeCode)

	for i, line := range inv.Lines {
		appendDocumentLine(root, "InvoiceLine", "InvoicedQuantity", i+1, line, inv.CurrencyCode, inv.DocumentTypeCode, domain.Identification{})
	}

	return doc, nil
}
