package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/application"
	"github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// Handler agrupa todos los handlers del módulo electronic.
type Handler struct {
	createDraft application.CreateDraftUseCase
	confirm     application.ConfirmUseCase
	get         application.GetDocumentUseCase
	list        application.ListDocumentsUseCase
	numbering   application.ManageNumberingUseCase
	pdf         *application.GetDocumentPDFUseCase
	fromSale    *application.CreateFromSaleUseCase
	audit       AuditLogger
}

func NewHandler(
	createDraft *application.CreateDraftUseCase,
	confirm *application.ConfirmUseCase,
	get *application.GetDocumentUseCase,
	list *application.ListDocumentsUseCase,
	numbering *application.ManageNumberingUseCase,
	pdf *application.GetDocumentPDFUseCase,
	fromSale *application.CreateFromSaleUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		createDraft: *createDraft,
		confirm:     *confirm,
		get:         *get,
		list:        *list,
		numbering:   *numbering,
		pdf:         pdf,
		fromSale:    fromSale,
		audit:       audit,
	}
}

// ── Documentos ────────────────────────────────────────────────────────────────────────────

func (h *Handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	docs, err := h.list.List(r.Context(), companyID, domain.ListFilter{Limit: 50})
	if err != nil {
		writeErr(w, err)
		return
	}
	if docs == nil {
		docs = []*domain.Document{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents": docs,
		"count":     len(docs),
	})
}

func (h *Handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.get.Get(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) handleConfirmDocument(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	doc, err := h.confirm.Confirm(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.confirmed", doc)
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) handleGetDocumentPDF(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	format := r.URL.Query().Get("format")
	pdfBytes, err := h.pdf.GetPDF(r.Context(), companyID, id, format)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="documento.pdf"`)
	_, _ = w.Write(pdfBytes)
}

func (h *Handler) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.createDraft.DeleteDraft(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "document.deleted", "document", &id, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Crear borradores ──────────────────────────────────────────────────────────────────────

type invoiceDraftBody struct {
	NumberingRangeID uuid.UUID        `json:"numbering_range_id"`
	Customer         cofdom.Party     `json:"customer"`
	Lines            []cofdom.Line    `json:"lines"`
	PaymentMeans     []cofdom.PaymentMean `json:"payment_means"`
	Note             string           `json:"note"`
	CurrencyCode     string           `json:"currency_code"`
	CustomerID       *uuid.UUID       `json:"customer_id"`
}

func (h *Handler) handleCreateInvoiceDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body invoiceDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CreateInvoiceDraft(r.Context(), application.InvoiceDraftRequest{
		CompanyID:        companyID,
		NumberingRangeID: body.NumberingRangeID,
		Customer:         body.Customer,
		Lines:            body.Lines,
		PaymentMeans:     body.PaymentMeans,
		Note:             body.Note,
		CurrencyCode:     body.CurrencyCode,
		CustomerID:       body.CustomerID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logDoc(r.Context(), companyID, "document.created", doc)
	writeJSON(w, http.StatusCreated, doc)
}

type noteDraftBody struct {
	NumberingRangeID    uuid.UUID                         `json:"numbering_range_id"`
	Customer            cofdom.Party                      `json:"customer"`
	Lines               []cofdom.Line                     `json:"lines"`
	PaymentMeans        []cofdom.PaymentMean              `json:"payment_means"`
	Note                string                            `json:"note"`
	CurrencyCode        string                            `json:"currency_code"`
	CustomerID          *uuid.UUID                        `json:"customer_id"`
	BillingReference    domain.BillingReferenceInput      `json:"billing_reference"`
	DiscrepancyResponse *domain.DiscrepancyResponseInput  `json:"discrepancy_response"`
	CreditNoteTypeCode  string                            `json:"credit_note_type_code"`
}

func (h *Handler) handleCreateCreditNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body noteDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CreateCreditNoteDraft(r.Context(), application.NoteDraftRequest{
		CompanyID: companyID, NumberingRangeID: body.NumberingRangeID,
		Customer: body.Customer, Lines: body.Lines, PaymentMeans: body.PaymentMeans,
		Note: body.Note, CurrencyCode: body.CurrencyCode, CustomerID: body.CustomerID,
		BillingReference: body.BillingReference, DiscrepancyResponse: body.DiscrepancyResponse,
		CreditNoteTypeCode: body.CreditNoteTypeCode,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (h *Handler) handleCreateDebitNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body noteDraftBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CreateDebitNoteDraft(r.Context(), application.NoteDraftRequest{
		CompanyID: companyID, NumberingRangeID: body.NumberingRangeID,
		Customer: body.Customer, Lines: body.Lines, PaymentMeans: body.PaymentMeans,
		Note: body.Note, CurrencyCode: body.CurrencyCode, CustomerID: body.CustomerID,
		BillingReference: body.BillingReference, DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

type supportDocBody struct {
	NumberingRangeID  uuid.UUID            `json:"numbering_range_id"`
	Vendor            cofdom.Party         `json:"vendor"`
	Lines             []cofdom.Line        `json:"lines"`
	PaymentMeans      []cofdom.PaymentMean `json:"payment_means"`
	Note              string               `json:"note"`
	CurrencyCode      string               `json:"currency_code"`
	OperationTypeCode string               `json:"operation_type_code"`
	WithholdingTaxes  []cofdom.Tax         `json:"withholding_taxes"`
	VendorID          *uuid.UUID           `json:"vendor_id"`
}

func (h *Handler) handleCreateSupportDocDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body supportDocBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CreateSupportDocumentDraft(r.Context(), application.SupportDocumentDraftRequest{
		CompanyID: companyID, NumberingRangeID: body.NumberingRangeID,
		Vendor: body.Vendor, Lines: body.Lines, PaymentMeans: body.PaymentMeans,
		Note: body.Note, CurrencyCode: body.CurrencyCode,
		OperationTypeCode: body.OperationTypeCode, WithholdingTaxes: body.WithholdingTaxes,
		VendorID: body.VendorID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

type adjustmentNoteBody struct {
	NumberingRangeID    uuid.UUID                        `json:"numbering_range_id"`
	Vendor              cofdom.Party                     `json:"vendor"`
	Lines               []cofdom.Line                    `json:"lines"`
	PaymentMeans        []cofdom.PaymentMean             `json:"payment_means"`
	Note                string                           `json:"note"`
	CurrencyCode        string                           `json:"currency_code"`
	OperationTypeCode   string                           `json:"operation_type_code"`
	WithholdingTaxes    []cofdom.Tax                     `json:"withholding_taxes"`
	VendorID            *uuid.UUID                       `json:"vendor_id"`
	BillingReference    domain.BillingReferenceInput     `json:"billing_reference"`
	DiscrepancyResponse *domain.DiscrepancyResponseInput `json:"discrepancy_response"`
}

func (h *Handler) handleCreateAdjustmentNoteDraft(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body adjustmentNoteBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.createDraft.CreateAdjustmentNoteDraft(r.Context(), application.AdjustmentNoteDraftRequest{
		CompanyID: companyID, NumberingRangeID: body.NumberingRangeID,
		Vendor: body.Vendor, Lines: body.Lines, PaymentMeans: body.PaymentMeans,
		Note: body.Note, CurrencyCode: body.CurrencyCode,
		OperationTypeCode: body.OperationTypeCode, WithholdingTaxes: body.WithholdingTaxes,
		VendorID: body.VendorID, BillingReference: body.BillingReference,
		DiscrepancyResponse: body.DiscrepancyResponse,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// ── Factura desde venta ───────────────────────────────────────────────────────────────────

type fromSaleBody struct {
	NumberingRangeID uuid.UUID `json:"numbering_range_id"`
}

func (h *Handler) handleCreateInvoiceFromSale(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(r.PathValue("sale_id"))
	if err != nil {
		http.Error(w, "sale_id inválido", http.StatusBadRequest)
		return
	}
	var body fromSaleBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	doc, err := h.fromSale.Execute(r.Context(), application.FromSaleRequest{
		CompanyID:        companyID,
		SaleID:           saleID,
		NumberingRangeID: body.NumberingRangeID,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

// ── Rangos de numeración ──────────────────────────────────────────────────────────────────

type numberingRangeBody struct {
	DianDocumentTypeCode string             `json:"dian_document_type_code"`
	Prefix               string             `json:"prefix"`
	ResolutionNumber     string             `json:"resolution_number"`
	ResolutionDate       string             `json:"resolution_date"` // YYYY-MM-DD
	RangeFrom            int64              `json:"range_from"`
	RangeTo              *int64             `json:"range_to"`
	ValidFrom            string             `json:"valid_from"` // YYYY-MM-DD
	ValidTo              string             `json:"valid_to"`
	Environment          domain.Environment `json:"environment"`
	TechnicalKey         string             `json:"technical_key"`
	TestSetID            string             `json:"test_set_id"`
}

func (h *Handler) handleCreateNumberingRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body numberingRangeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	nr := domain.NumberingRange{
		CompanyID:            companyID,
		DianDocumentTypeCode: body.DianDocumentTypeCode,
		Prefix:               body.Prefix,
		ResolutionNumber:     body.ResolutionNumber,
		RangeFrom:            body.RangeFrom,
		RangeTo:              body.RangeTo,
		Environment:          body.Environment,
		TechnicalKey:         body.TechnicalKey,
		TestSetID:            body.TestSetID,
		IsActive:             true,
	}
	if body.ResolutionDate != "" {
		if t, err := time.Parse("2006-01-02", body.ResolutionDate); err == nil {
			nr.ResolutionDate = t
		}
	}
	if body.ValidFrom != "" {
		if t, err := time.Parse("2006-01-02", body.ValidFrom); err == nil {
			nr.ValidFrom = t
		}
	}
	if body.ValidTo != "" {
		if t, err := time.Parse("2006-01-02", body.ValidTo); err == nil {
			nr.ValidTo = t
		}
	}
	if nr.RangeFrom <= 0 {
		nr.RangeFrom = 1
	}
	nr.CurrentNumber = nr.RangeFrom - 1
	created, err := h.numbering.Create(r.Context(), nr)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) handleListNumberingRanges(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	docType := r.URL.Query().Get("type")
	ranges, err := h.numbering.List(r.Context(), companyID, docType)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ranges)
}

func (h *Handler) handleDeactivateRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.numbering.Deactivate(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleActivateRange(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	nr, err := h.numbering.Activate(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nr)
}

// ── helpers ───────────────────────────────────────────────────────────────────────────────

// logDoc registra un evento de auditoría para una operación sobre un documento.
func (h *Handler) logDoc(ctx context.Context, companyID uuid.UUID, action string, doc *domain.Document) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	meta := map[string]any{
		"dian_document_type_code": doc.DianDocumentTypeCode,
	}
	if doc.Prefix != "" {
		meta["prefix"] = doc.Prefix
	}
	if doc.Number > 0 {
		meta["number"] = doc.Number
	}
	if doc.Customer.Name != "" {
		meta["customer_name"] = doc.Customer.Name
	} else if doc.Vendor != nil && doc.Vendor.Name != "" {
		meta["vendor_name"] = doc.Vendor.Name
	}
	h.audit.Log(ctx, companyID, userID, action, "document", &doc.ID, meta)
}

func mustTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id := tenant.GetCompanyID(r.Context())
	if id == uuid.Nil {
		http.Error(w, "empresa requerida", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrDocumentNotFound),
		errors.Is(err, domain.ErrRangeNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrDocumentNotDraft),
		errors.Is(err, domain.ErrEmptyLines),
		errors.Is(err, domain.ErrMissingCustomer),
		errors.Is(err, domain.ErrMissingPaymentMeans),
		errors.Is(err, domain.ErrMissingBillingReference),
		errors.Is(err, domain.ErrMissingVendor),
		errors.Is(err, domain.ErrRangeCompanyMismatch),
		errors.Is(err, domain.ErrWrongDocumentType),
		errors.Is(err, domain.ErrInvalidPaymentTerm),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidOperationTypeCode),
		errors.Is(err, domain.ErrRangeExhausted):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrCompanyNotReadyToIssue):
		status = http.StatusUnprocessableEntity
	}
	http.Error(w, err.Error(), status)
}
