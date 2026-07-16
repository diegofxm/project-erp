package builder

import (
	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendWithholdingTaxTotal agrega un cac:WithholdingTaxTotal por cada retención del
// Documento Soporte (InvoiceTypeCode "05"). A diferencia de cac:TaxTotal, donde todos los
// subtipos van bajo un mismo elemento, cada retención genera su propio
// cac:WithholdingTaxTotal con exactamente un cac:TaxSubtotal.
func appendWithholdingTaxTotal(parent *etree.Element, taxes []domain.Tax, currency string) {
	for _, t := range taxes {
		wtt := parent.CreateElement("cac:WithholdingTaxTotal")

		amt := wtt.CreateElement("cbc:TaxAmount")
		amt.CreateAttr("currencyID", currency)
		amt.SetText(formatAmount(t.TaxAmountCents))

		sub := wtt.CreateElement("cac:TaxSubtotal")

		taxable := sub.CreateElement("cbc:TaxableAmount")
		taxable.CreateAttr("currencyID", currency)
		taxable.SetText(formatAmount(t.TaxableAmountCents))

		taxAmt := sub.CreateElement("cbc:TaxAmount")
		taxAmt.CreateAttr("currencyID", currency)
		taxAmt.SetText(formatAmount(t.TaxAmountCents))

		category := sub.CreateElement("cac:TaxCategory")
		category.CreateElement("cbc:Percent").SetText(formatPercent(t.Percent))
		category.CreateElement("cbc:TaxExemptionReason").SetText("01")
		scheme := category.CreateElement("cac:TaxScheme")
		scheme.CreateElement("cbc:ID").SetText(t.TypeCode)
		scheme.CreateElement("cbc:Name").SetText(t.TypeName)
	}
}

// appendTaxTotal agrega cac:TaxTotal (cabecera o línea). No agrega nada si taxes está vacío,
// igual que exige el anexo técnico (TaxTotal es opcional cuando no hay impuestos aplicables).
func appendTaxTotal(parent *etree.Element, taxes []domain.Tax, currency string) {
	if len(taxes) == 0 {
		return
	}

	var totalCents int64
	for _, t := range taxes {
		totalCents += t.TaxAmountCents
	}

	taxTotal := parent.CreateElement("cac:TaxTotal")
	amt := taxTotal.CreateElement("cbc:TaxAmount")
	amt.CreateAttr("currencyID", currency)
	amt.SetText(formatAmount(totalCents))

	for _, t := range taxes {
		sub := taxTotal.CreateElement("cac:TaxSubtotal")

		taxable := sub.CreateElement("cbc:TaxableAmount")
		taxable.CreateAttr("currencyID", currency)
		taxable.SetText(formatAmount(t.TaxableAmountCents))

		taxAmt := sub.CreateElement("cbc:TaxAmount")
		taxAmt.CreateAttr("currencyID", currency)
		taxAmt.SetText(formatAmount(t.TaxAmountCents))

		category := sub.CreateElement("cac:TaxCategory")
		category.CreateElement("cbc:Percent").SetText(formatPercent(t.Percent))
		scheme := category.CreateElement("cac:TaxScheme")
		scheme.CreateElement("cbc:ID").SetText(t.TypeCode)
		scheme.CreateElement("cbc:Name").SetText(t.TypeName)
	}
}
