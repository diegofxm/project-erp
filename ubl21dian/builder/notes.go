package builder

import (
	"github.com/beevik/etree"
	"github.com/diegofxm/ubl21dian/domain"
)

// appendDiscrepancyResponse agrega cac:DiscrepancyResponse — el motivo de la nota. Solo se
// llama cuando la nota sí lo trae (operación "20" para Nota Crédito, "30" para Nota Débito).
func appendDiscrepancyResponse(parent *etree.Element, dr domain.DiscrepancyResponse) {
	el := parent.CreateElement("cac:DiscrepancyResponse")
	el.CreateElement("cbc:ReferenceID").SetText(dr.ReferenceID)
	el.CreateElement("cbc:ResponseCode").SetText(dr.ResponseCode)
	el.CreateElement("cbc:Description").SetText(dr.Description)
}

// appendBillingReference agrega cac:BillingReference — la referencia a la factura que esta
// nota corrige. node es "InvoiceDocumentReference" tanto para Nota Crédito como para Nota
// Débito.
func appendBillingReference(parent *etree.Element, node string, br domain.BillingReference) {
	ref := parent.CreateElement("cac:BillingReference").CreateElement("cac:" + node)
	ref.CreateElement("cbc:ID").SetText(br.Prefix + br.Number)
	uuid := ref.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeName", "CUFE-SHA384")
	uuid.SetText(br.CUFE)
	ref.CreateElement("cbc:IssueDate").SetText(br.IssueDate)
}
