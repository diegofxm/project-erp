package builder

import (
	"strconv"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendDocumentLine adds a detail line — shared by Invoice ("InvoiceLine"/
// "InvoicedQuantity"), CreditNote ("CreditNoteLine"/"CreditedQuantity") and DebitNote
// ("DebitNoteLine"/"DebitedQuantity"); node/nodeQty select which.
//
// For Invoice (documentTypeCode == "01") the Item carries StandardItemIdentification. For
// notes it instead carries InformationContentProviderParty/PowerOfAttorney/AgentParty with the
// "Mandante" (principal)'s identification — in practice, the issuer's own identification
// (mandanteID) when there is no distinct principal; verified against the technical annex
// (group FBA0x/CBA0x) and against a reference generator in real-world use.
// invoicePeriodStartDate: when not empty, adds cac:InvoicePeriod to the line — required on
// every line of the Support Document (DSFC01). Empty for Invoice/CreditNote/DebitNote.
func appendDocumentLine(parent *etree.Element, node, nodeQty string, index int, line domain.Line, currency, documentTypeCode string, mandanteID domain.Identification, invoicePeriodStartDate string) {
	el := parent.CreateElement("cac:" + node)
	el.CreateElement("cbc:ID").SetText(strconv.Itoa(index))

	qty := el.CreateElement("cbc:" + nodeQty)
	qty.CreateAttr("unitCode", line.UnitCode)
	qty.SetText(formatQuantity(line.Quantity))

	lineExt := el.CreateElement("cbc:LineExtensionAmount")
	lineExt.CreateAttr("currencyID", currency)
	lineExt.SetText(formatAmount(line.LineExtensionCents))

	if documentTypeCode != "92" {
		el.CreateElement("cbc:FreeOfChargeIndicator").SetText(formatBool(line.FreeOfCharge))
	}

	if line.FreeOfCharge && line.ReferencePrice != nil {
		altPrice := el.CreateElement("cac:PricingReference").CreateElement("cac:AlternativeConditionPrice")
		price := altPrice.CreateElement("cbc:PriceAmount")
		price.CreateAttr("currencyID", currency)
		price.SetText(formatAmount(line.ReferencePrice.PriceAmountCents))
		altPrice.CreateElement("cbc:PriceTypeCode").SetText(line.ReferencePrice.TypeCode)
	}

	if invoicePeriodStartDate != "" {
		ip := el.CreateElement("cac:InvoicePeriod")
		ip.CreateElement("cbc:StartDate").SetText(invoicePeriodStartDate)
		ip.CreateElement("cbc:DescriptionCode").SetText("1")
		ip.CreateElement("cbc:Description").SetText("Por operación")
	}

	appendTaxTotal(el, line.Taxes, currency)

	item := el.CreateElement("cac:Item")
	item.CreateElement("cbc:Description").SetText(line.Description)
	if documentTypeCode == "01" || documentTypeCode == "05" || documentTypeCode == "95" {
		sii := item.CreateElement("cac:StandardItemIdentification").CreateElement("cbc:ID")
		sii.CreateAttr("schemeID", line.ItemTypeCode)
		sii.CreateAttr("schemeName", line.ItemTypeName)
		// schemeAgencyID is only added when provided — code "999" (the taxpayer's own
		// standard, table 13.3.5 of the Technical Annex) explicitly requires this attribute be
		// omitted entirely, not even left empty (same rule as cac:Country in
		// appendAddressFields).
		if line.ItemTypeAgencyID != "" {
			sii.CreateAttr("schemeAgencyID", line.ItemTypeAgencyID)
		}
		sii.SetText(line.ItemCode)
	} else {
		agentID := item.CreateElement("cac:InformationContentProviderParty").
			CreateElement("cac:PowerOfAttorney").
			CreateElement("cac:AgentParty").
			CreateElement("cac:PartyIdentification").
			CreateElement("cbc:ID")
		setIdentificationAttrs(agentID, mandanteID)
		agentID.SetText(mandanteID.Number)
	}

	price := el.CreateElement("cac:Price")
	priceAmt := price.CreateElement("cbc:PriceAmount")
	priceAmt.CreateAttr("currencyID", currency)
	priceAmt.SetText(formatAmount(line.UnitPriceCents))
	baseQty := price.CreateElement("cbc:BaseQuantity")
	baseQty.CreateAttr("unitCode", line.UnitCode)
	baseQty.SetText(formatQuantity(line.Quantity))
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
