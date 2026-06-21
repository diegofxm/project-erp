package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/api-dian/internal/api/response"
	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/google/uuid"
)

// issueInvoiceRequest, issueCreditNoteRequest e issueDebitNoteRequest son los payloads
// públicos de emisión — Customer/Lines/PaymentMeans son pass-through puro (ver
// docs/api-dian-architecture.md sección 4.2): llegan tal cual y se persisten como snapshot.
//
// IssuerID/NumberingRangeID son string, no uuid.UUID: un uuid.UUID vacío o mal formado hace
// fallar el propio json.Decode (antes de llegar a validar nada), lo que se reportaba como el
// genérico "JSON inválido" sin decir cuál campo era — confuso de depurar (pasó de verdad
// probando con Postman). Decodificar como string siempre funciona; el UUID se valida después,
// campo por campo, con un mensaje claro (parseUUIDField).
type issueInvoiceRequest struct {
	IssuerID         string           `json:"issuer_id"`
	NumberingRangeID string           `json:"numbering_range_id"`
	Customer         partyDTO         `json:"customer"`
	Lines            []lineDTO        `json:"lines"`
	PaymentMeans     []paymentMeanDTO `json:"payment_means,omitempty"`
	Note             string           `json:"note,omitempty"`
	CurrencyCode     string           `json:"currency_code,omitempty"`
}

type issueNoteRequest struct {
	IssuerID            string                  `json:"issuer_id"`
	NumberingRangeID    string                  `json:"numbering_range_id"`
	Customer            partyDTO                `json:"customer"`
	Lines               []lineDTO               `json:"lines"`
	PaymentMeans        []paymentMeanDTO        `json:"payment_means,omitempty"`
	Note                string                  `json:"note,omitempty"`
	CurrencyCode        string                  `json:"currency_code,omitempty"`
	BillingReference    billingReferenceDTO     `json:"billing_reference"`
	DiscrepancyResponse *discrepancyResponseDTO `json:"discrepancy_response,omitempty"`
}

type issueCreditNoteRequest struct {
	issueNoteRequest
	CreditNoteTypeCode string `json:"credit_note_type_code"`
}

// documentResponse es la representación pública de un documento emitido — incluye el XML
// firmado completo (retención legal) y el estado de la DIAN, no solo el de api-dian.
type documentResponse struct {
	ID                    uuid.UUID `json:"id"`
	IssuerID              uuid.UUID `json:"issuer_id"`
	NumberingRangeID      uuid.UUID `json:"numbering_range_id"`
	DianDocumentTypeCode  string    `json:"dian_document_type_code"`
	Prefix                string    `json:"prefix"`
	Number                int64     `json:"number"`
	DocumentKey           string    `json:"document_key"`
	IssueDate             time.Time `json:"issue_date"`
	QRURL                 string    `json:"qr_url"`
	SignedXML             string    `json:"signed_xml"`
	Status                string    `json:"status"`
	DianTrackID           string    `json:"dian_track_id,omitempty"`
	DianStatusCode        string    `json:"dian_status_code,omitempty"`
	DianStatusDescription string    `json:"dian_status_description,omitempty"`
	DianStatusMessage     string    `json:"dian_status_message,omitempty"`
}

func documentToResponse(d *documents.Document) documentResponse {
	return documentResponse{
		ID:                    d.ID,
		IssuerID:              d.IssuerID,
		NumberingRangeID:      d.NumberingRangeID,
		DianDocumentTypeCode:  d.DianDocumentTypeCode,
		Prefix:                d.Prefix,
		Number:                d.Number,
		DocumentKey:           d.DocumentKey,
		IssueDate:             d.IssueDate,
		QRURL:                 d.QRURL,
		SignedXML:             d.SignedXML,
		Status:                string(d.Status),
		DianTrackID:           d.DianTrackID,
		DianStatusCode:        d.DianStatusCode,
		DianStatusDescription: d.DianStatusDescription,
		DianStatusMessage:     d.DianStatusMessage,
	}
}

func (a *API) handleIssueInvoice(w http.ResponseWriter, r *http.Request) {
	var req issueInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	issuerID, ok := parseUUIDField(w, req.IssuerID, "issuer_id")
	if !ok {
		return
	}
	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return
	}

	doc, err := a.documents.IssueInvoice(r.Context(), documents.IssueInvoiceRequest{
		IssuerID:         issuerID,
		NumberingRangeID: rangeID,
		Customer:         req.Customer.toDomain(),
		Lines:            linesToDomain(req.Lines),
		PaymentMeans:     paymentMeansToDomain(req.PaymentMeans),
		Note:             req.Note,
		CurrencyCode:     req.CurrencyCode,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleIssueCreditNote(w http.ResponseWriter, r *http.Request) {
	var req issueCreditNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	noteReq, ok := toServiceNoteRequest(w, req.issueNoteRequest)
	if !ok {
		return
	}

	doc, err := a.documents.IssueCreditNote(r.Context(), documents.IssueCreditNoteRequest{
		IssueNoteRequest:   noteReq,
		CreditNoteTypeCode: req.CreditNoteTypeCode,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleIssueDebitNote(w http.ResponseWriter, r *http.Request) {
	var req issueNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	noteReq, ok := toServiceNoteRequest(w, req)
	if !ok {
		return
	}

	doc, err := a.documents.IssueDebitNote(r.Context(), noteReq)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	doc, err := a.documents.GetDocument(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func toServiceNoteRequest(w http.ResponseWriter, req issueNoteRequest) (documents.IssueNoteRequest, bool) {
	issuerID, ok := parseUUIDField(w, req.IssuerID, "issuer_id")
	if !ok {
		return documents.IssueNoteRequest{}, false
	}
	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return documents.IssueNoteRequest{}, false
	}

	out := documents.IssueNoteRequest{
		IssuerID:         issuerID,
		NumberingRangeID: rangeID,
		Customer:         req.Customer.toDomain(),
		Lines:            linesToDomain(req.Lines),
		PaymentMeans:     paymentMeansToDomain(req.PaymentMeans),
		Note:             req.Note,
		CurrencyCode:     req.CurrencyCode,
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
