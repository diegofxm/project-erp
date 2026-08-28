package builder

import (
	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendMonetaryTotal adds the totals block — shared by Invoice, CreditNote and DebitNote.
// node is "LegalMonetaryTotal" for Invoice/CreditNote, "RequestedMonetaryTotal" for DebitNote
// (same content, different element name — verified in section 8.4 of the technical annex).
func appendMonetaryTotal(parent *etree.Element, node string, t domain.Totals, currency, documentTypeCode string) {
	el := parent.CreateElement("cac:" + node)

	line := el.CreateElement("cbc:LineExtensionAmount")
	line.CreateAttr("currencyID", currency)
	line.SetText(formatAmount(t.LineExtensionCents))

	taxExcl := el.CreateElement("cbc:TaxExclusiveAmount")
	taxExcl.CreateAttr("currencyID", currency)
	taxExcl.SetText(formatAmount(t.TaxExclusiveCents))

	taxIncl := el.CreateElement("cbc:TaxInclusiveAmount")
	taxIncl.CreateAttr("currencyID", currency)
	taxIncl.SetText(formatAmount(t.TaxInclusiveCents))

	if documentTypeCode == "01" && t.PrepaidCents > 0 {
		prepaid := el.CreateElement("cbc:PrepaidAmount")
		prepaid.CreateAttr("currencyID", currency)
		prepaid.SetText(formatAmount(t.PrepaidCents))
	}

	payable := el.CreateElement("cbc:PayableAmount")
	payable.CreateAttr("currencyID", currency)
	payable.SetText(formatAmount(t.PayableCents))
}
