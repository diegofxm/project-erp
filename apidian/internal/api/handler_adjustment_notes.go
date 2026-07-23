package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/documents"
)

// issueAdjustmentNoteRequest es el payload público de la Nota de Ajuste al DS.
// Combina los campos del DS (vendor, retenciones, tipo operación) con la referencia
// obligatoria al DS original (billing_reference con CUDS) y el motivo opcional
// (discrepancy_response). La empresa emisora/compradora se deriva del token — no se acepta del cliente.
type issueAdjustmentNoteRequest struct {
	NumberingRangeID    string                  `json:"numbering_range_id"`
	VendorID            string                  `json:"vendor_id,omitempty"`
	Vendor              partyDTO                `json:"vendor"`
	Lines               []lineInputDTO          `json:"lines"`
	PaymentMeans        []paymentMeanDTO        `json:"payment_means,omitempty"`
	Note                string                  `json:"note,omitempty"`
	CurrencyCode        string                  `json:"currency_code,omitempty"`
	OperationTypeCode   string                  `json:"operation_type_code"` // "10" Residente / "11" No Residente
	WithholdingTaxes    []taxDTO                `json:"withholding_taxes,omitempty"`
	BillingReference    billingReferenceDTO     `json:"billing_reference"`
	DiscrepancyResponse *discrepancyResponseDTO `json:"discrepancy_response,omitempty"`
}

func (a *API) handleCreateAdjustmentNote(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeIssueAdjustmentNoteRequest(w, r)
	if !ok {
		return
	}
	doc, err := a.documents.CreateAdjustmentNoteDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	vendorName := ""
	if doc.Vendor != nil {
		vendorName = doc.Vendor.Name
	}
	a.logEvent(r, "document.created", "document", &doc.ID, map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
		"vendor_name":             vendorName,
	})
	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleUpdateAdjustmentNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	req, ok := decodeIssueAdjustmentNoteRequest(w, r)
	if !ok {
		return
	}
	doc, err := a.documents.UpdateAdjustmentNoteDraft(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func decodeIssueAdjustmentNoteRequest(w http.ResponseWriter, r *http.Request) (documents.IssueAdjustmentNoteRequest, bool) {
	var req issueAdjustmentNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return documents.IssueAdjustmentNoteRequest{}, false
	}
	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return documents.IssueAdjustmentNoteRequest{}, false
	}
	vendorID, ok := parseOptionalUUIDField(w, req.VendorID, "vendor_id")
	if !ok {
		return documents.IssueAdjustmentNoteRequest{}, false
	}
	out := documents.IssueAdjustmentNoteRequest{
		IssuerID:          middleware.GetTenantID(r.Context()),
		NumberingRangeID:  rangeID,
		VendorID:          vendorID,
		Vendor:            req.Vendor.toDomain(),
		Lines:             linesToInput(req.Lines),
		PaymentMeans:      paymentMeansToDomain(req.PaymentMeans),
		Note:              req.Note,
		CurrencyCode:      req.CurrencyCode,
		OperationTypeCode: req.OperationTypeCode,
		WithholdingTaxes:  withholdingTaxesToDomain(req.WithholdingTaxes),
		BillingReference: documents.BillingReferenceInput{
			Prefix:    req.BillingReference.Prefix,
			Number:    req.BillingReference.Number,
			CUFE:      req.BillingReference.CUFE,
			IssueDate: req.BillingReference.IssueDate,
		},
	}
	if req.DiscrepancyResponse != nil {
		out.DiscrepancyResponse = &documents.DiscrepancyResponseInput{
			ReferenceID:  req.DiscrepancyResponse.ReferenceID,
			ResponseCode: req.DiscrepancyResponse.ResponseCode,
			Description:  req.DiscrepancyResponse.Description,
		}
	}
	return out, true
}
