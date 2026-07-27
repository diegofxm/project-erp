package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/edocuments/documents"
	"github.com/diegofxm/apidian/internal/pdf"
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
	Lines            []lineInputDTO   `json:"lines"`
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
	Lines               []lineInputDTO          `json:"lines"`
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
// que el usuario ya capturó. Prefix/Number/DocumentKey/IssueDate/QRURL son omitempty: vacíos
// mientras Status == "draft", porque todavía no se reclamó número ni se firmó.
// El XML firmado NO se incluye aquí — se descarga por separado via GET /documents/{id}/xml.
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

	// Solo Documento Soporte (DianDocumentTypeCode "05") — nil/vacíos en los demás.
	Vendor            *partyDTO `json:"vendor,omitempty"`
	OperationTypeCode string    `json:"operation_type_code,omitempty"`
	WithholdingTaxes  []taxDTO  `json:"withholding_taxes,omitempty"`

	// Solo se llenan al confirmar (POST /documents/{id}/confirm) — vacíos mientras Status ==
	// "draft". SignedXML no se incluye: se sirve por GET /documents/{id}/xml (ver
	// handleGetDocumentXML), igual que el PDF tiene su propio endpoint.
	Prefix      string     `json:"prefix,omitempty"`
	Number      int64      `json:"number,omitempty"`
	DocumentKey string     `json:"document_key,omitempty"`
	IssueDate   *time.Time `json:"issue_date,omitempty"`
	QRURL       string     `json:"qr_url,omitempty"`

	DianTrackID           string `json:"dian_track_id,omitempty"`
	DianStatusCode        string `json:"dian_status_code,omitempty"`
	DianStatusDescription string `json:"dian_status_description,omitempty"`
	DianStatusMessage     string `json:"dian_status_message,omitempty"`

	// CustomerID es solo trazabilidad — ver documents.Document.CustomerID. nil si el
	// documento no referenció un cliente guardado.
	CustomerID *uuid.UUID `json:"customer_id,omitempty"`
	// VendorID es solo trazabilidad para DS — nil si el DS no se creó desde un vendor guardado.
	VendorID *uuid.UUID `json:"vendor_id,omitempty"`

	// NCCount/NDCount: cuántas NC/ND referencian esta factura. Solo presentes (>0) en el
	// listado de facturas; siempre 0 (omitido) en GET /documents/{id} individual.
	NCCount int `json:"nc_count,omitempty"`
	NDCount int `json:"nd_count,omitempty"`

	// RelatedNotes/NetPayableCents: solo presentes en GET /documents/{id} para FE (01) y DS
	// (05) confirmados — vacíos en borradores, en NC/ND/NA y en el listado de documentos.
	RelatedNotes    []relatedNoteDTO `json:"related_notes,omitempty"`
	NetPayableCents *int64           `json:"net_payable_cents,omitempty"`

	// SourceDocumentID: ID del FE o DS al que esta NC/ND/NA hace referencia. Solo presente en
	// NC (91), ND (92) y NA (95) — se resuelve buscando por el CUFE del billing_reference.
	SourceDocumentID *uuid.UUID `json:"source_document_id,omitempty"`

	// CreatedAt/UpdatedAt — mismo criterio que customerResponse/productResponse. Un borrador
	// no tiene IssueDate todavía, así que es lo único con lo que un listado puede ordenar u
	// ofrecer "creado hace X" antes de confirmar.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// relatedNoteDTO es la vista pública de un NC/ND/NA relacionado con un documento.
type relatedNoteDTO struct {
	ID                   uuid.UUID  `json:"id"`
	DianDocumentTypeCode string     `json:"dian_document_type_code"`
	Prefix               string     `json:"prefix,omitempty"`
	Number               int64      `json:"number,omitempty"`
	PayableCents         int64      `json:"payable_cents"`
	Status               string     `json:"status"`
	IssueDate            *time.Time `json:"issue_date,omitempty"`
}

func relatedNoteToDTO(n documents.RelatedNote) relatedNoteDTO {
	return relatedNoteDTO{
		ID:                   n.ID,
		DianDocumentTypeCode: n.DianDocumentTypeCode,
		Prefix:               n.Prefix,
		Number:               n.Number,
		PayableCents:         n.PayableCents,
		Status:               string(n.Status),
		IssueDate:            n.IssueDate,
	}
}

// computeNetPayable calcula el saldo neto de una FE o DS dado el conjunto de notas relacionadas.
// Para FE: payable − Σ NC aceptadas + Σ ND aceptadas.
// Para DS: payable − Σ NA aceptadas.
func computeNetPayable(doc *documents.Document, notes []documents.RelatedNote) *int64 {
	net := doc.Totals.PayableCents
	for _, n := range notes {
		if n.Status != documents.StatusAccepted {
			continue
		}
		switch n.DianDocumentTypeCode {
		case "91": // NC — reduce saldo
			net -= n.PayableCents
		case "92": // ND — aumenta saldo
			net += n.PayableCents
		case "95": // NA — reduce saldo del DS
			net -= n.PayableCents
		}
	}
	return &net
}

func documentToResponse(d *documents.Document) documentResponse {
	resp := documentResponse{
		ID:                    d.ID,
		IssuerID:              d.IssuerID,
		NumberingRangeID:      d.NumberingRangeID,
		CustomerID:            d.CustomerID,
		VendorID:              d.VendorID,
		DianDocumentTypeCode:  d.DianDocumentTypeCode,
		Status:                string(d.Status),
		Customer:              partyFromDomain(d.Customer),
		Lines:                 linesFromDomain(d.Lines),
		PaymentMeans:          paymentMeansFromDomain(d.PaymentMeans),
		Totals:                totalsFromDomain(d.Totals),
		Note:                  d.Note,
		CurrencyCode:          d.CurrencyCode,
		NoteTypeCode:          d.NoteTypeCode,
		OperationTypeCode:     d.OperationTypeCode,
		Prefix:                d.Prefix,
		Number:                d.Number,
		DocumentKey:           d.DocumentKey,
		QRURL:                 d.QRURL,
		DianTrackID:           d.DianTrackID,
		DianStatusCode:        d.DianStatusCode,
		DianStatusDescription: d.DianStatusDescription,
		DianStatusMessage:     d.DianStatusMessage,
		NCCount:               d.NCCount,
		NDCount:               d.NDCount,
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
	if d.Vendor != nil {
		v := partyFromDomain(*d.Vendor)
		resp.Vendor = &v
	}
	if len(d.WithholdingTaxes) > 0 {
		resp.WithholdingTaxes = make([]taxDTO, len(d.WithholdingTaxes))
		for i, t := range d.WithholdingTaxes {
			resp.WithholdingTaxes[i] = taxFromDomain(t)
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
	a.logEvent(r, "document.created", "document", &doc.ID, map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
		"customer_name":           doc.Customer.Name,
	})
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
		Lines:            linesToInput(req.Lines),
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
	a.logEvent(r, "document.created", "document", &doc.ID, map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
		"customer_name":           doc.Customer.Name,
	})
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
	a.logEvent(r, "document.created", "document", &doc.ID, map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
		"customer_name":           doc.Customer.Name,
	})
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
		Lines:            linesToInput(req.Lines),
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

// confirmDocumentRequest es el body opcional de POST /documents/{id}/confirm.
// Solo es relevante para DS ("05") y NA ("95") — los demás tipos lo ignoran.
type confirmDocumentRequest struct {
	// ExpenseAccountCode es el código PUC del gasto o costo (ej. "5135" Servicios,
	// "5120" Arrendamientos). Requerido para que el asiento contable de DS/NA sea correcto;
	// si se omite, el asiento DS/NA no se registra y se emite un aviso en logs.
	ExpenseAccountCode string `json:"expense_account_code"`
}

// handleConfirmDocument reclama el consecutivo real, construye, firma, y —si el ambiente lo
// permite— envía un borrador ya creado. Único punto donde se "gasta" un número real de la
// DIAN — antes de esto, el documento se podía editar o eliminar libremente (ver sección 9.25
// del architecture doc).
func (a *API) handleConfirmDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var body confirmDocumentRequest
	_ = json.NewDecoder(r.Body).Decode(&body) // EOF cuando no hay body — aceptable

	doc, err := a.documents.ConfirmDocument(r.Context(), middleware.GetTenantID(r.Context()), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	a.logEvent(r, "document.confirmed", "document", &doc.ID, map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
		"prefix":                  doc.Prefix,
		"number":                  doc.Number,
		"customer_name":           doc.Customer.Name,
	})

	a.postAccountingEntry(r, doc, body.ExpenseAccountCode)

	response.WriteJSON(w, http.StatusOK, documentToResponse(doc))
}

// postAccountingEntry registra el asiento contable correspondiente al documento confirmado.
// Es fire-and-forget: los errores se loggean pero nunca fallan la respuesta HTTP.
func (a *API) postAccountingEntry(r *http.Request, doc *documents.Document, expenseAccountCode string) {
	if a.accountingClient == nil {
		return
	}
	companyID := middleware.GetTenantID(r.Context())
	ctx := r.Context()
	var err error
	switch doc.DianDocumentTypeCode {
	case "01": // FE — Factura de venta
		err = a.accountingClient.PostInvoice(ctx, doc, companyID)
	case "91": // NC — Nota Crédito
		err = a.accountingClient.PostCreditNote(ctx, doc, companyID)
	case "92": // ND — Nota Débito
		err = a.accountingClient.PostDebitNote(ctx, doc, companyID)
	case "05": // DS — Documento Soporte
		if expenseAccountCode == "" {
			a.log.Sugar().Warnf("accounting: DS %s confirmado sin expense_account_code — asiento omitido", doc.ID)
			return
		}
		err = a.accountingClient.PostSupportDocument(ctx, doc, companyID, expenseAccountCode)
	case "95": // NA — Nota de Ajuste al DS
		if expenseAccountCode == "" {
			a.log.Sugar().Warnf("accounting: NA %s confirmada sin expense_account_code — asiento omitido", doc.ID)
			return
		}
		err = a.accountingClient.PostAdjustmentNote(ctx, doc, companyID, expenseAccountCode)
	}
	if err != nil {
		a.log.Sugar().Warnf("accounting: asiento %s %s%d: %v", doc.DianDocumentTypeCode, doc.Prefix, doc.Number, err)
	}
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
	a.logEvent(r, "document.deleted", "document", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

// handleCloneDocument crea un borrador nuevo copiando customer/lines/payment_means/note del
// documento fuente. Solo aplica a Facturas Electrónicas (tipo "01") — las notas (NC/ND) y
// Documentos Soporte (DS) llevan referencias obligatorias al original que no tiene sentido
// clonar. La clonación conserva el mismo NumberingRangeID; si el rango fue desactivado, el
// error lo reporta CreateInvoiceDraft igual que si el usuario creara una factura nueva.
func (a *API) handleCloneDocument(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	issuerID := middleware.GetTenantID(r.Context())

	doc, err := a.documents.GetDocument(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if doc.IssuerID != issuerID {
		response.WriteError(w, documents.ErrDocumentNotFound)
		return
	}
	if doc.DianDocumentTypeCode != "01" {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "solo se pueden clonar facturas electrónicas (tipo 01)"})
		return
	}

	lines := make([]documents.LineInput, len(doc.Lines))
	for i, l := range doc.Lines {
		li := documents.LineInput{
			Description:    l.Description,
			Quantity:       l.Quantity,
			UnitCode:       l.UnitCode,
			UnitPriceCents: l.UnitPriceCents,
			ItemCode:       l.ItemCode,
			ItemTypeCode:   l.ItemTypeCode,
		}
		if len(l.Taxes) > 0 {
			li.TaxTypeCode = l.Taxes[0].TypeCode
			li.TaxPercent = l.Taxes[0].Percent
		}
		lines[i] = li
	}

	req := documents.IssueInvoiceRequest{
		IssuerID:         issuerID,
		NumberingRangeID: doc.NumberingRangeID,
		Customer:         doc.Customer,
		Lines:            lines,
		PaymentMeans:     doc.PaymentMeans,
		Note:             doc.Note,
		CurrencyCode:     doc.CurrencyCode,
		CustomerID:       doc.CustomerID,
	}

	clone, err := a.documents.CreateInvoiceDraft(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	a.logEvent(r, "document.cloned", "document", &clone.ID, map[string]any{
		"source_document_id": id.String(),
		"customer_name":      clone.Customer.Name,
	})
	response.WriteJSON(w, http.StatusCreated, documentToResponse(clone))
}

// ── Listado / consulta ───────────────────────────────────────────────────────────────────────

// handleListDocuments devuelve los documentos del emisor autenticado, opcionalmente
// filtrados por ?dian_document_type_code=&status=&from=&to= (fechas YYYY-MM-DD, sobre
// issue_date), ?source_document_id=<uuid> (NC/ND que referencian esa factura), y paginados
// con ?limit=&offset= (normalizados en documents.Service.ListDocuments si se omiten o
// exceden el máximo).
func (a *API) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := documents.ListFilter{
		DianDocumentTypeCode: q.Get("dian_document_type_code"),
		Status:               documents.Status(q.Get("status")),
	}

	if s := q.Get("source_document_id"); s != "" {
		id, ok := parseUUID(w, s)
		if !ok {
			return
		}
		filter.SourceDocumentID = &id
	}

	if s := q.Get("q"); s != "" {
		filter.Search = s
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
	issuerID := middleware.GetTenantID(r.Context())
	if doc.IssuerID != issuerID {
		response.WriteError(w, documents.ErrDocumentNotFound)
		return
	}

	resp := documentToResponse(doc)

	// Para FE (01) y DS (05) confirmados: adjuntar notas relacionadas y saldo neto.
	if doc.Number != 0 && (doc.DianDocumentTypeCode == "01" || doc.DianDocumentTypeCode == "05") {
		notes, err := a.documents.GetRelatedNotes(r.Context(), issuerID, doc.Prefix, doc.Number)
		if err == nil {
			dtos := make([]relatedNoteDTO, len(notes))
			for i, n := range notes {
				dtos[i] = relatedNoteToDTO(n)
			}
			resp.RelatedNotes = dtos
			resp.NetPayableCents = computeNetPayable(doc, notes)
		}
	}

	// Para NC (91), ND (92) y NA (95): resolver el ID del documento de origen por CUFE.
	if (doc.DianDocumentTypeCode == "91" || doc.DianDocumentTypeCode == "92" || doc.DianDocumentTypeCode == "95") &&
		doc.BillingReference != nil && doc.BillingReference.CUFE != "" {
		if srcDoc, err := a.documents.GetDocumentByDocumentKey(r.Context(), issuerID, doc.BillingReference.CUFE); err == nil {
			resp.SourceDocumentID = &srcDoc.ID
		}
	}

	response.WriteJSON(w, http.StatusOK, resp)
}

// handleGetDocumentPDF sirve la representación gráfica en PDF — generada en memoria en cada
// petición, nunca se guarda a disco (ver docs/apidian-architecture.md sección 9.39/9.49). Sirve
// igual para Factura, NC y ND, ya sea borrador o confirmado.
// Acepta ?format=full_a4 (defecto) o ?format=half_a4 (dos copias por hoja con línea de corte).
func (a *API) handleGetDocumentPDF(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	format := pdf.ParseFormat(r.URL.Query().Get("format"))
	pdfBytes, err := a.documents.RenderDocumentPDF(r.Context(), middleware.GetTenantID(r.Context()), id, format)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="documento.pdf"`)
	_, _ = w.Write(pdfBytes)
}

// handleGetDocumentXML sirve el XML UBL firmado — disponible solo después de que el documento
// fue confirmado (POST /documents/{id}/confirm). Para borradores (SignedXML vacío) devuelve
// 409/ErrDocumentNotSigned. Sigue el mismo patrón que handleGetDocumentPDF: endpoint de
// descarga separado en lugar de embeberlo en el JSON (el XML puede pesar 100-200 KB).
func (a *API) handleGetDocumentXML(w http.ResponseWriter, r *http.Request) {
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
	if doc.SignedXML == "" {
		response.WriteError(w, documents.ErrDocumentNotSigned)
		return
	}

	filename := strconv.FormatInt(doc.Number, 10) + ".xml"
	if doc.Prefix != "" {
		filename = doc.Prefix + filename
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte(doc.SignedXML))
}

type sendEmailRequest struct {
	PDFFormat string   `json:"pdf_format,omitempty"`
	To        string   `json:"to,omitempty"` // override del destinatario; vacío = resolver desde catálogo
	CC        []string `json:"cc,omitempty"`
}

// handleSendDocumentEmail envía el documento accepted al correo del cliente/proveedor con el PDF
// y el XML firmado adjuntos. Body opcional: {"to":"email","pdf_format":"half_a4","cc":[]}.
func (a *API) handleSendDocumentEmail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var req sendEmailRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
			return
		}
	}

	format := pdf.ParseFormat(req.PDFFormat)
	if err := a.documents.SendDocumentEmail(r.Context(), middleware.GetTenantID(r.Context()), id, format, req.To, req.CC); err != nil {
		response.WriteError(w, err)
		return
	}
	a.logEvent(r, "document.email_sent", "document", &id, nil)
	w.WriteHeader(http.StatusNoContent)
}

type verifyAcquirerResponse struct {
	Found         bool   `json:"found"`
	ReceiverName  string `json:"receiver_name,omitempty"`
	ReceiverEmail string `json:"receiver_email,omitempty"`
}

// handleVerifyAcquirer es una ayuda OPCIONAL al capturar un NIT (?identification_type_code=&
// identification_number=) — nunca bloqueante, ver documents.Service.VerifyAcquirer y
// docs/apidian-architecture.md sección 9.41. found=false es el resultado normal y esperado
// para la mayoría de cédulas, no un error.
func (a *API) handleVerifyAcquirer(w http.ResponseWriter, r *http.Request) {
	typeCode := r.URL.Query().Get("identification_type_code")
	number := r.URL.Query().Get("identification_number")
	if typeCode == "" || number == "" {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "identification_type_code e identification_number son obligatorios"})
		return
	}

	info, err := a.documents.VerifyAcquirer(r.Context(), middleware.GetTenantID(r.Context()), typeCode, number)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, verifyAcquirerResponse{
		Found:         info.Found,
		ReceiverName:  info.ReceiverName,
		ReceiverEmail: info.ReceiverEmail,
	})
}

type dianRangeItem struct {
	ResolutionNumber     string `json:"resolution_number"`
	ResolutionDate       string `json:"resolution_date"`
	Prefix               string `json:"prefix"`
	RangeFrom            int64  `json:"range_from"`
	RangeTo              int64  `json:"range_to"`
	ValidFrom            string `json:"valid_from"`
	ValidTo              string `json:"valid_to"`
	TechnicalKey         string `json:"technical_key,omitempty"`
	SuggestedDocTypeCode string `json:"suggested_doc_type_code,omitempty"`
}

type getDianNumberingRangesResponse struct {
	Ranges []dianRangeItem `json:"ranges"`
}

// handleGetDianNumberingRanges consulta los rangos de numeración del emisor activo ante la
// DIAN vía GetNumberingRange — devuelve los datos tal como los reporta la DIAN, con una
// sugerencia de tipo de documento cuando el prefijo es reconocible (SETP/SEDS). El frontend
// usa estos datos para pre-llenar el formulario de importación; el usuario aporta
// dian_document_type_code y, si el ambiente es habilitación, el test_set_id.
func (a *API) handleGetDianNumberingRanges(w http.ResponseWriter, r *http.Request) {
	ranges, err := a.documents.GetDianNumberingRanges(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	items := make([]dianRangeItem, 0, len(ranges))
	for _, dr := range ranges {
		items = append(items, dianRangeItem{
			ResolutionNumber:     dr.ResolutionNumber,
			ResolutionDate:       dr.ResolutionDate,
			Prefix:               dr.Prefix,
			RangeFrom:            dr.RangeFrom,
			RangeTo:              dr.RangeTo,
			ValidFrom:            dr.ValidFrom,
			ValidTo:              dr.ValidTo,
			TechnicalKey:         dr.TechnicalKey,
			SuggestedDocTypeCode: dr.SuggestedDocTypeCode,
		})
	}

	response.WriteJSON(w, http.StatusOK, getDianNumberingRangesResponse{Ranges: items})
}
