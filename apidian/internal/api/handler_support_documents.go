package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/diegofxm/cofacture/domain"
)

// issueSupportDocumentRequest es el payload público del Documento Soporte. Vendor es el
// tercero no obligado (AccountingSupplierParty); la empresa emisora actúa de compradora y
// se deriva del token — no se acepta del cliente. OperationTypeCode distingue Residente
// ("10") de No Residente ("11").
type issueSupportDocumentRequest struct {
	NumberingRangeID  string           `json:"numbering_range_id"`
	Vendor            partyDTO         `json:"vendor"`
	Lines             []lineInputDTO   `json:"lines"`
	PaymentMeans      []paymentMeanDTO `json:"payment_means,omitempty"`
	Note              string           `json:"note,omitempty"`
	CurrencyCode      string           `json:"currency_code,omitempty"`
	OperationTypeCode string           `json:"operation_type_code"` // "10" Residente / "11" No Residente
	WithholdingTaxes  []taxDTO         `json:"withholding_taxes,omitempty"`
}

func (a *API) handleCreateSupportDocument(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeIssueSupportDocumentRequest(w, r)
	if !ok {
		return
	}
	doc, err := a.documents.CreateSupportDocumentDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleUpdateSupportDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	req, ok := decodeIssueSupportDocumentRequest(w, r)
	if !ok {
		return
	}
	doc, err := a.documents.UpdateSupportDocumentDraft(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func decodeIssueSupportDocumentRequest(w http.ResponseWriter, r *http.Request) (documents.IssueSupportDocumentRequest, bool) {
	var req issueSupportDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return documents.IssueSupportDocumentRequest{}, false
	}
	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return documents.IssueSupportDocumentRequest{}, false
	}
	return documents.IssueSupportDocumentRequest{
		IssuerID:          middleware.GetTenantID(r.Context()),
		NumberingRangeID:  rangeID,
		Vendor:            req.Vendor.toDomain(),
		Lines:             linesToInput(req.Lines),
		PaymentMeans:      paymentMeansToDomain(req.PaymentMeans),
		Note:              req.Note,
		CurrencyCode:      req.CurrencyCode,
		OperationTypeCode: req.OperationTypeCode,
		WithholdingTaxes:  withholdingTaxesToDomain(req.WithholdingTaxes),
	}, true
}

func withholdingTaxesToDomain(dtos []taxDTO) []domain.Tax {
	if len(dtos) == 0 {
		return nil
	}
	out := make([]domain.Tax, len(dtos))
	for i, t := range dtos {
		out[i] = domain.Tax{
			TaxableAmountCents: t.TaxableAmountCents,
			TaxAmountCents:     t.TaxAmountCents,
			Percent:            t.Percent,
			TypeCode:           t.TypeCode,
			TypeName:           t.TypeName,
		}
	}
	return out
}
