package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/diegofxm/api-dian/internal/api/middleware"
	"github.com/diegofxm/api-dian/internal/api/response"
	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/google/uuid"
)

// issueInvoiceRequest, issueCreditNoteRequest e issueDebitNoteRequest son los payloads
// públicos de emisión — Customer/Lines/PaymentMeans son pass-through puro (ver
// docs/api-dian-architecture.md sección 4.2): llegan tal cual y se persisten como snapshot.
//
// IssuerID NO es un campo del body — siempre es el emisor del usuario autenticado
// (middleware.GetTenantID), nunca algo que el cliente pueda elegir; así un usuario nunca
// puede emitir documentos a nombre de otro emisor.
//
// NumberingRangeID es string, no uuid.UUID: un uuid.UUID vacío o mal formado hace fallar el
// propio json.Decode (antes de llegar a validar nada), lo que se reportaba como el genérico
// "JSON inválido" sin decir cuál campo era — confuso de depurar (pasó de verdad probando con
// Postman). Decodificar como string siempre funciona; el UUID se valida después con un
// mensaje claro (parseUUIDField). documents.Service.prepare además verifica que el rango
// pertenezca a este mismo emisor (ErrNumberingRangeIssuerMismatch).
type issueInvoiceRequest struct {
	NumberingRangeID string           `json:"numbering_range_id"`
	Customer         partyDTO         `json:"customer"`
	Lines            []lineDTO        `json:"lines"`
	PaymentMeans     []paymentMeanDTO `json:"payment_means,omitempty"`
	Note             string           `json:"note,omitempty"`
	CurrencyCode     string           `json:"currency_code,omitempty"`

	// CustomerID es opcional — referencia de solo trazabilidad a un cliente guardado en
	// internal/customers (ver documents.IssueInvoiceRequest.CustomerID). NUNCA reemplaza
	// "customer": el snapshot pass-through sigue siendo obligatorio y es lo que se firma.
	// documents.Service.prepare verifica que pertenezca a este mismo emisor
	// (ErrCustomerIssuerMismatch), igual criterio que numbering_range_id.
	CustomerID string `json:"customer_id,omitempty"`
}

type issueNoteRequest struct {
	NumberingRangeID    string                  `json:"numbering_range_id"`
	Customer            partyDTO                `json:"customer"`
	Lines               []lineDTO               `json:"lines"`
	PaymentMeans        []paymentMeanDTO        `json:"payment_means,omitempty"`
	Note                string                  `json:"note,omitempty"`
	CurrencyCode        string                  `json:"currency_code,omitempty"`
	BillingReference    billingReferenceDTO     `json:"billing_reference"`
	DiscrepancyResponse *discrepancyResponseDTO `json:"discrepancy_response,omitempty"`
	CustomerID          string                  `json:"customer_id,omitempty"` // opcional, ver issueInvoiceRequest.CustomerID
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

	// CustomerID es solo trazabilidad — ver documents.Document.CustomerID. nil si la factura
	// no referenció un cliente guardado.
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
}

func documentToResponse(d *documents.Document) documentResponse {
	return documentResponse{
		ID:                    d.ID,
		IssuerID:              d.IssuerID,
		NumberingRangeID:      d.NumberingRangeID,
		CustomerID:            d.CustomerID,
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

	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return
	}
	customerID, ok := parseOptionalUUIDField(w, req.CustomerID, "customer_id")
	if !ok {
		return
	}

	doc, err := a.documents.IssueInvoice(r.Context(), documents.IssueInvoiceRequest{
		IssuerID:         middleware.GetTenantID(r.Context()),
		NumberingRangeID: rangeID,
		Customer:         req.Customer.toDomain(),
		Lines:            linesToDomain(req.Lines),
		PaymentMeans:     paymentMeansToDomain(req.PaymentMeans),
		Note:             req.Note,
		CurrencyCode:     req.CurrencyCode,
		CustomerID:       customerID,
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

	noteReq, ok := toServiceNoteRequest(w, middleware.GetTenantID(r.Context()), req.issueNoteRequest)
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

	noteReq, ok := toServiceNoteRequest(w, middleware.GetTenantID(r.Context()), req)
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

// handleListDocuments devuelve los documentos del emisor autenticado, opcionalmente
// filtrados por ?dian_document_type_code=&status=&from=&to= (fechas YYYY-MM-DD, sobre
// issue_date) y paginados con ?limit=&offset= (normalizados en documents.Service.ListDocuments
// si se omiten o exceden el máximo).
func (a *API) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := documents.ListFilter{
		DianDocumentTypeCode: q.Get("dian_document_type_code"),
		Status:               documents.Status(q.Get("status")),
	}

	if s := q.Get("from"); s != "" {
		from, err := parseDate(s)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "from inválida, se espera YYYY-MM-DD"})
			return
		}
		filter.From = from
	}
	if s := q.Get("to"); s != "" {
		to, err := parseDate(s)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "to inválida, se espera YYYY-MM-DD"})
			return
		}
		filter.To = to
	}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "limit inválido, se espera un entero positivo"})
			return
		}
		filter.Limit = n
	}
	if s := q.Get("offset"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "offset inválido, se espera un entero positivo"})
			return
		}
		filter.Offset = n
	}

	docs, err := a.documents.ListDocuments(r.Context(), middleware.GetTenantID(r.Context()), filter)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]documentResponse, len(docs))
	for i, d := range docs {
		out[i] = documentToResponse(d)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"documents": out, "count": len(out)})
}

// handleGetDocument exige que el documento pertenezca al emisor autenticado — si no, se
// responde el mismo 404 que un documento inexistente (documents.ErrDocumentNotFound), para no
// revelarle a un usuario que el ID que probó existe pero es de otro emisor.
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
	if doc.IssuerID != middleware.GetTenantID(r.Context()) {
		response.WriteError(w, documents.ErrDocumentNotFound)
		return
	}

	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func toServiceNoteRequest(w http.ResponseWriter, issuerID uuid.UUID, req issueNoteRequest) (documents.IssueNoteRequest, bool) {
	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return documents.IssueNoteRequest{}, false
	}
	customerID, ok := parseOptionalUUIDField(w, req.CustomerID, "customer_id")
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
		CustomerID:       customerID,
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
