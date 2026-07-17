package builder

import (
	"strconv"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendDocumentLine agrega un renglón de detalle — compartido por Invoice ("InvoiceLine"/
// "InvoicedQuantity"), CreditNote ("CreditNoteLine"/"CreditedQuantity") y DebitNote
// ("DebitNoteLine"/"DebitedQuantity"); node/nodeQty eligen cuál.
//
// Para Invoice (documentTypeCode == "01") el Item lleva StandardItemIdentification. Para
// notas lleva en su lugar InformationContentProviderParty/PowerOfAttorney/AgentParty con la
// identificación del "Mandante" — en la práctica, la del propio emisor (mandanteID) cuando
// no hay un mandante distinto; verificado contra el anexo técnico (grupo FBA0x/CBA0x) y
// contra un generador de referencia en uso real.
func appendDocumentLine(parent *etree.Element, node, nodeQty string, index int, line domain.Line, currency, documentTypeCode string, mandanteID domain.Identification) {
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

	appendTaxTotal(el, line.Taxes, currency)

	item := el.CreateElement("cac:Item")
	item.CreateElement("cbc:Description").SetText(line.Description)
	if documentTypeCode == "01" || documentTypeCode == "05" {
		sii := item.CreateElement("cac:StandardItemIdentification").CreateElement("cbc:ID")
		sii.CreateAttr("schemeID", line.ItemTypeCode)
		sii.CreateAttr("schemeName", line.ItemTypeName)
		// schemeAgencyID solo se agrega si viene informado — la fila "999" (estándar propio
		// del contribuyente, tabla 13.3.5 del Anexo Técnico) exige explícitamente que este
		// atributo NO se use, ni siquiera vacío (mismo criterio que cac:Country en
		// appendAddressFields, sección 9.44/9.45).
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
