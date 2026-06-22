package builder

import (
	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendPaymentMean agrega cac:PaymentMeans — compartido por Invoice, CreditNote y
// DebitNote (mismo elemento, mismo contenido en los tres).
func appendPaymentMean(parent *etree.Element, pm domain.PaymentMean) {
	el := parent.CreateElement("cac:PaymentMeans")
	el.CreateElement("cbc:ID").SetText(pm.Code)
	el.CreateElement("cbc:PaymentMeansCode").SetText(pm.PaymentMethodCode)
	if pm.Code == "2" {
		el.CreateElement("cbc:PaymentDueDate").SetText(pm.DueDate)
	}
	if pm.PaymentReference != "" {
		el.CreateElement("cbc:PaymentID").SetText(pm.PaymentReference)
	}
}
