package builder

import (
	"strconv"
	"strings"

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

	// DS: DocumentCurrencyCode sin atributos de lista (DSFC03 — el esquema DS los rechaza).
	root.CreateElement("cbc:DocumentCurrencyCode").SetText(inv.CurrencyCode)

	root.CreateElement("cbc:LineCountNumeric").SetText(strconv.Itoa(len(inv.Lines)))

	// Roles invertidos: Supplier = no-obligado, Customer = empresa compradora/emisora.
	// DS usa estructuras de partes más simples que FE — funciones dedicadas por tipo.
	appendDSSupplierParty(root, inv.Supplier)
	appendDSCustomerParty(root, inv.Customer)

	for _, pm := range inv.PaymentMeans {
		appendPaymentMean(root, pm)
	}

	appendTaxTotal(root, inv.HeaderTaxes, inv.CurrencyCode)
	appendWithholdingTaxTotal(root, inv.WithholdingTaxes, inv.CurrencyCode)
	appendMonetaryTotal(root, "LegalMonetaryTotal", inv.Totals, inv.CurrencyCode, inv.DocumentTypeCode)

	// Fecha de periodo por línea: PeriodStartDate del documento si se especificó, o IssueDate.
	// cac:InvoicePeriod es obligatorio en cada InvoiceLine del DS (DSFC01).
	linePeriodDate := inv.PeriodStartDate
	if linePeriodDate == "" {
		linePeriodDate = inv.IssueDate
	}
	for i, line := range inv.Lines {
		appendDocumentLine(root, "InvoiceLine", "InvoicedQuantity", i+1, line, inv.CurrencyCode, inv.DocumentTypeCode, domain.Identification{}, linePeriodDate)
	}

	return doc, nil
}

// appendDSSupplierParty genera el AccountingSupplierParty del Documento Soporte.
// La DIAN exige schemeName="31" (NIT) para el CompanyID del SNO en todos los casos —
// incluyendo personas naturales — según comportamiento verificado en DS real aceptado por la
// DIAN (DS-real.xml). TaxLevelCode va sin atributo listName (también verificado en DS real).
func appendDSSupplierParty(root *etree.Element, p domain.Party) {
	ap := root.CreateElement("cac:AccountingSupplierParty")
	ap.CreateElement("cbc:AdditionalAccountID").SetText(p.EntityTypeCode)
	party := ap.CreateElement("cac:Party")

	party.CreateElement("cac:PartyName").CreateElement("cbc:Name").SetText(p.Name)

	if p.Address.Line != "" {
		addr := party.CreateElement("cac:PhysicalLocation").CreateElement("cac:Address")
		appendAddressFields(addr, p.Address)
	}

	taxScheme := party.CreateElement("cac:PartyTaxScheme")
	taxScheme.CreateElement("cbc:RegistrationName").SetText(p.Name)
	companyID := taxScheme.CreateElement("cbc:CompanyID")
	companyID.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	companyID.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	// La DIAN requiere schemeName="31" (NIT) para el SNO; schemeID es el dígito verificador.
	if p.Identification.VerificationCode != "" {
		companyID.CreateAttr("schemeID", p.Identification.VerificationCode)
	}
	companyID.CreateAttr("schemeName", "31")
	companyID.SetText(p.Identification.Number)
	if len(p.LiabilityCodes) > 0 {
		taxScheme.CreateElement("cbc:TaxLevelCode").SetText(strings.Join(p.LiabilityCodes, ";"))
	}
	scheme := taxScheme.CreateElement("cac:TaxScheme")
	scheme.CreateElement("cbc:ID").SetText(p.TaxSchemeCode)
	scheme.CreateElement("cbc:Name").SetText(p.TaxSchemeName)
}

// appendDSCustomerParty genera el AccountingCustomerParty del Documento Soporte.
// La empresa compradora/emisora lleva SOLO PartyTaxScheme — sin PartyIdentification,
// PhysicalLocation, RegistrationAddress, PartyLegalEntity ni Contact.
// TaxLevelCode NO lleva el atributo listName.
// Verificado contra la caja de herramientas DIAN DS v1.1.
func appendDSCustomerParty(root *etree.Element, p domain.Party) {
	ap := root.CreateElement("cac:AccountingCustomerParty")
	ap.CreateElement("cbc:AdditionalAccountID").SetText(p.EntityTypeCode)
	party := ap.CreateElement("cac:Party")

	party.CreateElement("cac:PartyName").CreateElement("cbc:Name").SetText(p.Name)

	if p.Address.Line != "" {
		addr := party.CreateElement("cac:PhysicalLocation").CreateElement("cac:Address")
		appendAddressFields(addr, p.Address)
	}

	taxScheme := party.CreateElement("cac:PartyTaxScheme")
	taxScheme.CreateElement("cbc:RegistrationName").SetText(p.Name)
	companyID := taxScheme.CreateElement("cbc:CompanyID")
	setIdentificationAttrs(companyID, p.Identification)
	companyID.SetText(p.Identification.Number)
	if len(p.LiabilityCodes) > 0 {
		taxScheme.CreateElement("cbc:TaxLevelCode").SetText(strings.Join(p.LiabilityCodes, ";"))
	}
	scheme := taxScheme.CreateElement("cac:TaxScheme")
	scheme.CreateElement("cbc:ID").SetText(p.TaxSchemeCode)
	scheme.CreateElement("cbc:Name").SetText(p.TaxSchemeName)
}
