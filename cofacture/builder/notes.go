package builder

import (
	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
)

// appendDiscrepancyResponse adds cac:DiscrepancyResponse — the note's reason. Only called when
// the note actually carries one (operation "20" for Credit Note, "30" for Debit Note).
func appendDiscrepancyResponse(parent *etree.Element, dr domain.DiscrepancyResponse) {
	el := parent.CreateElement("cac:DiscrepancyResponse")
	el.CreateElement("cbc:ReferenceID").SetText(dr.ReferenceID)
	el.CreateElement("cbc:ResponseCode").SetText(dr.ResponseCode)
	el.CreateElement("cbc:Description").SetText(dr.Description)
}

// appendBillingReference adds cac:BillingReference — the reference to the invoice this note
// corrects. node is "InvoiceDocumentReference" for both Credit Note and Debit Note.
func appendBillingReference(parent *etree.Element, node string, br domain.BillingReference) {
	ref := parent.CreateElement("cac:BillingReference").CreateElement("cac:" + node)
	ref.CreateElement("cbc:ID").SetText(br.Prefix + br.Number)
	uuid := ref.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeName", "CUFE-SHA384")
	uuid.SetText(br.CUFE)
	ref.CreateElement("cbc:IssueDate").SetText(br.IssueDate)
}
