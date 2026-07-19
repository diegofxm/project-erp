package builder

import (
	"strconv"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
	ubl "github.com/diegofxm/cofacture/xml"
)

// BuildAdjustmentNote construye el árbol XML de una Nota de Ajuste al Documento Soporte
// (InvoiceTypeCode "95"). Es al DS (tipo 05) lo que NC/ND son a la Factura: usa el mismo
// elemento raíz Invoice, los mismos roles invertidos (Supplier=SNO, Customer=ABS), y el
// mismo algoritmo CUDS-SHA384. Añade BillingReference al DS original y DiscrepancyResponse.
func BuildAdjustmentNote(an domain.AdjustmentNote) (*etree.Document, error) {
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

	// appendUBLExtensions maneja InvoiceControl para tipo "05"/"95" (familia DS).
	appendUBLExtensions(root, an.Invoice)

	root.CreateElement("cbc:UBLVersionID").SetText("UBL 2.1")
	root.CreateElement("cbc:CustomizationID").SetText(an.OperationTypeCode)
	root.CreateElement("cbc:ProfileID").SetText(an.ProfileID)
	root.CreateElement("cbc:ProfileExecutionID").SetText(an.EnvironmentCode)
	root.CreateElement("cbc:ID").SetText(an.Prefix + an.Number)

	uuid := root.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeID", an.EnvironmentCode)
	uuid.CreateAttr("schemeName", an.HashType) // "CUDS-SHA384"
	uuid.SetText(an.CUFE)

	root.CreateElement("cbc:IssueDate").SetText(an.IssueDate)
	root.CreateElement("cbc:IssueTime").SetText(an.IssueTime)
	root.CreateElement("cbc:InvoiceTypeCode").SetText(an.DocumentTypeCode) // "95"
	if an.Note != "" {
		root.CreateElement("cbc:Note").SetText(an.Note)
	}

	// NA: DocumentCurrencyCode sin atributos de lista (igual que DS — DSFC03).
	root.CreateElement("cbc:DocumentCurrencyCode").SetText(an.CurrencyCode)

	root.CreateElement("cbc:LineCountNumeric").SetText(strconv.Itoa(len(an.Lines)))

	// DiscrepancyResponse va antes de BillingReference (mismo orden que NC/ND).
	if an.DiscrepancyResponse != nil {
		appendDiscrepancyResponse(root, *an.DiscrepancyResponse)
	}

	// BillingReference apunta al DS original — el UUID lleva schemeName="CUDS-SHA384"
	// (a diferencia de NC/ND que referencian una FE y usan "CUFE-SHA384").
	appendNABillingReference(root, an.BillingReference)

	// Roles invertidos iguales al DS: Supplier = tercero no obligado, Customer = ABS.
	appendDSSupplierParty(root, an.Supplier)
	appendDSCustomerParty(root, an.Customer)

	for _, pm := range an.PaymentMeans {
		appendPaymentMean(root, pm)
	}

	appendTaxTotal(root, an.HeaderTaxes, an.CurrencyCode)
	appendWithholdingTaxTotal(root, an.WithholdingTaxes, an.CurrencyCode)
	appendMonetaryTotal(root, "LegalMonetaryTotal", an.Totals, an.CurrencyCode, an.DocumentTypeCode)

	linePeriodDate := an.PeriodStartDate
	if linePeriodDate == "" {
		linePeriodDate = an.IssueDate
	}
	for i, line := range an.Lines {
		appendDocumentLine(root, "InvoiceLine", "InvoicedQuantity", i+1, line, an.CurrencyCode, an.DocumentTypeCode, domain.Identification{}, linePeriodDate)
	}

	return doc, nil
}

// appendNABillingReference agrega cac:BillingReference al DS original con schemeName
// "CUDS-SHA384" — distinto de NC/ND que referencian una FE con "CUFE-SHA384".
func appendNABillingReference(parent *etree.Element, br domain.BillingReference) {
	ref := parent.CreateElement("cac:BillingReference").CreateElement("cac:InvoiceDocumentReference")
	ref.CreateElement("cbc:ID").SetText(br.Prefix + br.Number)
	uuid := ref.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeName", "CUDS-SHA384")
	uuid.SetText(br.CUFE)
	ref.CreateElement("cbc:IssueDate").SetText(br.IssueDate)
}
