package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/google/uuid"
)

// issueInvoiceRequest, issueCreditNoteRequest e issueNoteRequest son los payloads públicos
// de Invoice/CreditNote/DebitNote — sirven tanto para crear un borrador (POST) como para
// reemplazarlo por completo (PUT, solo mientras siga en borrador). Customer/Lines/
// PaymentMeans son pass-through puro (ver docs/apidian-architecture.md sección 4.2): llegan
// tal cual y se persisten como snapshot — nunca se reclama número, firma ni envía aquí, eso
// solo pasa en POST /documents/{id}/confirm (ver sección 9.25).
//
// IssuerID NO es un campo del body — siempre es el emisor del usuario autenticado
// (middleware.GetTenantID), nunca algo que el cliente pueda elegir; así un usuario nunca
// puede crear/editar documentos a nombre de otro emisor.
//
// NumberingRangeID es string, no uuid.UUID: un uuid.UUID vacío o mal formado hace fallar el
// propio json.Decode (antes de llegar a validar nada), lo que se reportaba como el genérico
// "JSON inválido" sin decir cuál campo era — confuso de depurar (pasó de verdad probando con
// Postman). Decodificar como string siempre funciona; el UUID se valida después con un
// mensaje claro (parseUUIDField). documents.Service.validateForIssuance además verifica que
// el rango pertenezca a este mismo emisor (ErrNumberingRangeIssuerMismatch).
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
	// documents.Service.validateForIssuance verifica que pertenezca a este mismo emisor
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

// documentResponse es la representación pública de un documento. Customer/Lines/PaymentMeans/
// Totals/Note/CurrencyCode están SIEMPRE presentes, incluso en borrador — son el contenido
// que el usuario ya capturó. Prefix/Number/DocumentKey/IssueDate/QRURL/SignedXML son
// punteros/omitempty: vacíos mientras Status == "draft", porque todavía no se reclamó número
// ni se firmó (ver documents.Document).
//
// Customer/Lines/PaymentMeans/Totals se agregaron en la Fase 2.28 — antes GET /documents y
// GET /documents/{id} solo devolvían metadatos (ID, status, fechas), nunca el contenido en
// sí, así que un frontend no podía mostrar "factura para Juan Pérez por $119.000" sin volver
// a pedirle los datos al usuario. Se encontró construyendo el dashboard real (ver
// docs/apidian-architecture.md sección 9.28) — exactamente el tipo de hueco que probar la
// API como un usuario real, no solo con Postman, sí revela.
type documentResponse struct {
	ID                   uuid.UUID `json:"id"`
	IssuerID             uuid.UUID `json:"issuer_id"`
	NumberingRangeID     uuid.UUID `json:"numbering_range_id"`
	DianDocumentTypeCode string    `json:"dian_document_type_code"`
	Status               string    `json:"status"`

	Customer     partyDTO         `json:"customer"`
	Lines        []lineDTO        `json:"lines"`
	PaymentMeans []paymentMeanDTO `json:"payment_means,omitempty"`
	Totals       totalsDTO        `json:"totals"`
	Note         string           `json:"note,omitempty"`
	CurrencyCode string           `json:"currency_code,omitempty"`

	// Solo CreditNote/DebitNote — nil en Invoice.
	BillingReference    *billingReferenceDTO    `json:"billing_reference,omitempty"`
	DiscrepancyResponse *discrepancyResponseDTO `json:"discrepancy_response,omitempty"`
	NoteTypeCode        string                  `json:"note_type_code,omitempty"`

	// Solo se llenan al confirmar (POST /documents/{id}/confirm) — vacíos mientras Status ==
	// "draft".
	Prefix      string     `json:"prefix,omitempty"`
	Number      int64      `json:"number,omitempty"`
	DocumentKey string     `json:"document_key,omitempty"`
	IssueDate   *time.Time `json:"issue_date,omitempty"`
	QRURL       string     `json:"qr_url,omitempty"`
	SignedXML   string     `json:"signed_xml,omitempty"`

	DianTrackID           string `json:"dian_track_id,omitempty"`
	DianStatusCode        string `json:"dian_status_code,omitempty"`
	DianStatusDescription string `json:"dian_status_description,omitempty"`
	DianStatusMessage     string `json:"dian_status_message,omitempty"`

	// CustomerID es solo trazabilidad — ver documents.Document.CustomerID. nil si el
	// documento no referenció un cliente guardado.
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`

	// CreatedAt/UpdatedAt — mismo criterio que customerResponse/productResponse. Un borrador
	// no tiene IssueDate todavía, así que es lo único con lo que un listado puede ordenar u
	// ofrecer "creado hace X" antes de confirmar.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func documentToResponse(d *documents.Document) documentResponse {
	resp := documentResponse{
		ID:                    d.ID,
		IssuerID:              d.IssuerID,
		NumberingRangeID:      d.NumberingRangeID,
		CustomerID:            d.CustomerID,
		DianDocumentTypeCode:  d.DianDocumentTypeCode,
		Status:                string(d.Status),
		Customer:              partyFromDomain(d.Customer),
		Lines:                 linesFromDomain(d.Lines),
		PaymentMeans:          paymentMeansFromDomain(d.PaymentMeans),
		Totals:                totalsFromDomain(d.Totals),
		Note:                  d.Note,
		CurrencyCode:          d.CurrencyCode,
		NoteTypeCode:          d.NoteTypeCode,
		Prefix:                d.Prefix,
		Number:                d.Number,
		DocumentKey:           d.DocumentKey,
		QRURL:                 d.QRURL,
		SignedXML:             d.SignedXML,
		DianTrackID:           d.DianTrackID,
		DianStatusCode:        d.DianStatusCode,
		DianStatusDescription: d.DianStatusDescription,
		DianStatusMessage:     d.DianStatusMessage,
		CreatedAt:             d.CreatedAt,
		UpdatedAt:             d.UpdatedAt,
	}
	if !d.IssueDate.IsZero() {
		resp.IssueDate = &d.IssueDate
	}
	if d.BillingReference != nil {
		resp.BillingReference = &billingReferenceDTO{
			Prefix:    d.BillingReference.Prefix,
			Number:    d.BillingReference.Number,
			CUFE:      d.BillingReference.CUFE,
			IssueDate: d.BillingReference.IssueDate,
		}
	}
	if d.DiscrepancyResponse != nil {
		resp.DiscrepancyResponse = &discrepancyResponseDTO{
			ReferenceID:  d.DiscrepancyResponse.ReferenceID,
			ResponseCode: d.DiscrepancyResponse.ResponseCode,
			Description:  d.DiscrepancyResponse.Description,
		}
	}
	return resp
}

// ── Invoice ──────────────────────────────────────────────────────────────────────────────────

func (a *API) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeIssueInvoiceRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.CreateInvoiceDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	req, ok := decodeIssueInvoiceRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.UpdateInvoiceDraft(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func decodeIssueInvoiceRequest(w http.ResponseWriter, r *http.Request) (documents.IssueInvoiceRequest, bool) {
	var req issueInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return documents.IssueInvoiceRequest{}, false
	}

	rangeID, ok := parseUUIDField(w, req.NumberingRangeID, "numbering_range_id")
	if !ok {
		return documents.IssueInvoiceRequest{}, false
	}
	customerID, ok := parseOptionalUUIDField(w, req.CustomerID, "customer_id")
	if !ok {
		return documents.IssueInvoiceRequest{}, false
	}

	return documents.IssueInvoiceRequest{
		IssuerID:         middleware.GetTenantID(r.Context()),
		NumberingRangeID: rangeID,
		Customer:         req.Customer.toDomain(),
		Lines:            linesToDomain(req.Lines),
		PaymentMeans:     paymentMeansToDomain(req.PaymentMeans),
		Note:             req.Note,
		CurrencyCode:     req.CurrencyCode,
		CustomerID:       customerID,
	}, true
}

// ── Credit Note ──────────────────────────────────────────────────────────────────────────────

func (a *API) handleCreateCreditNote(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeIssueCreditNoteRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.CreateCreditNoteDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleUpdateCreditNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	req, ok := decodeIssueCreditNoteRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.UpdateCreditNoteDraft(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func decodeIssueCreditNoteRequest(w http.ResponseWriter, r *http.Request) (documents.IssueCreditNoteRequest, bool) {
	var req issueCreditNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return documents.IssueCreditNoteRequest{}, false
	}

	noteReq, ok := toServiceNoteRequest(w, middleware.GetTenantID(r.Context()), req.issueNoteRequest)
	if !ok {
		return documents.IssueCreditNoteRequest{}, false
	}

	return documents.IssueCreditNoteRequest{
		IssueNoteRequest:   noteReq,
		CreditNoteTypeCode: req.CreditNoteTypeCode,
	}, true
}

// ── Debit Note ───────────────────────────────────────────────────────────────────────────────

func (a *API) handleCreateDebitNote(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeIssueNoteRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.CreateDebitNoteDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, documentToResponse(doc))
}

func (a *API) handleUpdateDebitNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	req, ok := decodeIssueNoteRequest(w, r)
	if !ok {
		return
	}

	doc, err := a.documents.UpdateDebitNoteDraft(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

func decodeIssueNoteRequest(w http.ResponseWriter, r *http.Request) (documents.IssueNoteRequest, bool) {
	var req issueNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return documents.IssueNoteRequest{}, false
	}
	return toServiceNoteRequest(w, middleware.GetTenantID(r.Context()), req)
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

// ── Confirmar / eliminar (compartido por los tres tipos) ───────────────────────────────────

// handleConfirmDocument reclama el consecutivo real, construye, firma, y —si el ambiente lo
// permite— envía un borrador ya creado. Único punto donde se "gasta" un número real de la
// DIAN — antes de esto, el documento se podía editar o eliminar libremente (ver sección 9.25
// del architecture doc).
func (a *API) handleConfirmDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	doc, err := a.documents.ConfirmDocument(r.Context(), middleware.GetTenantID(r.Context()), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

// handleDeleteDocument elimina un borrador — solo mientras siga en borrador
// (documents.ErrDocumentNotDraft si ya fue confirmado) y pertenezca al emisor autenticado.
func (a *API) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := a.documents.DeleteDraft(r.Context(), middleware.GetTenantID(r.Context()), id); err != nil {
		response.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Listado / consulta ───────────────────────────────────────────────────────────────────────

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
