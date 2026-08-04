package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/application"
	"github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// ── DTOs de salida ──────────────────────────────────────────────────────────────────────────
// domain.PurchaseOrder/PurchaseLine no llevan tags json (no deben — el dominio no conoce HTTP) —
// acá se mapean a snake_case, igual que sales/interfaces/http (mismo bug real que se encontró
// ahí: antes este handler mandaba el struct de dominio crudo, con campos en PascalCase).

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatDate(*t)
}

type purchaseLineDTO struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	TaxRate     float64   `json:"tax_rate"`
	Subtotal    float64   `json:"subtotal"`
	TaxAmount   float64   `json:"tax_amount"`
	Total       float64   `json:"total"`
}

type withholdingDTO struct {
	ID              uuid.UUID `json:"id"`
	PurchaseOrderID uuid.UUID `json:"purchase_order_id"`
	ConceptCode     string    `json:"concept_code"`
	ConceptName     string    `json:"concept_name"`
	Base            float64   `json:"base"`
	RateBP          int       `json:"rate_bp"`
	Amount          float64   `json:"amount"`
	AccountPayable  string    `json:"account_payable"`
	CreatedAt       time.Time `json:"created_at"`
}

func toWithholdingDTO(w domain.PurchaseWithholding) withholdingDTO {
	return withholdingDTO{
		ID: w.ID, PurchaseOrderID: w.PurchaseOrderID, ConceptCode: w.ConceptCode, ConceptName: w.ConceptName,
		Base: w.Base, RateBP: w.RateBP, Amount: w.Amount, AccountPayable: w.AccountPayable, CreatedAt: w.CreatedAt,
	}
}

type purchaseDTO struct {
	ID                uuid.UUID         `json:"id"`
	CompanyID         uuid.UUID         `json:"company_id"`
	SupplierID        uuid.UUID         `json:"supplier_id"`
	Number            string            `json:"number"`
	Status            string            `json:"status"`
	IssueDate         string            `json:"issue_date"`
	DueDate           string            `json:"due_date,omitempty"`
	Notes             string            `json:"notes"`
	Lines             []purchaseLineDTO `json:"lines"`
	Withholdings      []withholdingDTO  `json:"withholdings"`
	SupportDocumentID *uuid.UUID        `json:"support_document_id,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func toPurchaseDTO(o *domain.PurchaseOrder) purchaseDTO {
	lines := make([]purchaseLineDTO, len(o.Lines))
	for i, l := range o.Lines {
		lines[i] = purchaseLineDTO{
			ID: l.ID, ProductID: l.ProductID, Description: l.Description,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, Discount: l.Discount, TaxRate: l.TaxRate,
			Subtotal: l.Subtotal, TaxAmount: l.TaxAmount, Total: l.Total,
		}
	}
	withholdings := make([]withholdingDTO, len(o.Withholdings))
	for i, wh := range o.Withholdings {
		withholdings[i] = toWithholdingDTO(wh)
	}
	return purchaseDTO{
		ID: o.ID, CompanyID: o.CompanyID, SupplierID: o.SupplierID,
		Number: o.Number, Status: string(o.Status),
		IssueDate: formatDate(o.IssueDate), DueDate: formatDatePtr(o.DueDate),
		Notes: o.Notes, Lines: lines, Withholdings: withholdings, SupportDocumentID: o.SupportDocumentID,
		CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt,
	}
}

type paymentDTO struct {
	ID            uuid.UUID `json:"id"`
	CompanyID     uuid.UUID `json:"company_id"`
	PurchaseID    uuid.UUID `json:"purchase_id"`
	PaymentDate   string    `json:"payment_date"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Reference     string    `json:"reference"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPaymentDTO(p *domain.PurchasePayment) paymentDTO {
	return paymentDTO{
		ID: p.ID, CompanyID: p.CompanyID, PurchaseID: p.PurchaseID,
		PaymentDate: formatDate(p.PaymentDate), Amount: p.Amount,
		PaymentMethod: string(p.PaymentMethod), Reference: p.Reference, Notes: p.Notes,
		CreatedAt: p.CreatedAt,
	}
}

type payableDTO struct {
	PurchaseID     uuid.UUID `json:"purchase_id"`
	PurchaseNumber string    `json:"purchase_number"`
	SupplierID     uuid.UUID `json:"supplier_id"`
	IssueDate      string    `json:"issue_date"`
	DueDate        string    `json:"due_date,omitempty"`
	Total          float64   `json:"total"`
	Paid           float64   `json:"paid"`
	Balance        float64   `json:"balance"`
}

func toPayableDTO(b domain.PayableBalance) payableDTO {
	return payableDTO{
		PurchaseID: b.PurchaseID, PurchaseNumber: b.PurchaseNumber, SupplierID: b.SupplierID,
		IssueDate: formatDate(b.IssueDate), DueDate: formatDatePtr(b.DueDate),
		Total: b.Total, Paid: b.Paid, Balance: b.Balance,
	}
}

type Handler struct {
	create      *application.CreateUseCase
	get         *application.GetUseCase
	confirm     *application.ConfirmUseCase
	cancel      *application.CancelUseCase
	receive     *application.ReceiveUseCase
	delete      *application.DeleteUseCase
	payment     *application.PaymentUseCase
	pdf         *application.GetPurchaseOrderPDFUseCase
	sendEmail   *application.SendPurchaseOrderEmailUseCase
	withholding *application.AddWithholdingUseCase
	audit       AuditLogger
}

func NewHandler(
	create *application.CreateUseCase,
	get *application.GetUseCase,
	confirm *application.ConfirmUseCase,
	cancel *application.CancelUseCase,
	receive *application.ReceiveUseCase,
	del *application.DeleteUseCase,
	payment *application.PaymentUseCase,
	pdf *application.GetPurchaseOrderPDFUseCase,
	sendEmail *application.SendPurchaseOrderEmailUseCase,
	withholding *application.AddWithholdingUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		create: create, get: get, confirm: confirm, cancel: cancel, receive: receive, delete: del, payment: payment,
		pdf: pdf, sendEmail: sendEmail, withholding: withholding, audit: audit,
	}
}

func (h *Handler) logAudit(ctx context.Context, companyID uuid.UUID, action, resourceType string, id uuid.UUID, meta map[string]any) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	h.audit.Log(ctx, companyID, userID, action, resourceType, &id, meta)
}

func (h *Handler) handleAddWithholding(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		ConceptID uuid.UUID `json:"concept_id"`
		Base      float64   `json:"base"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	wh, err := h.withholding.Execute(r.Context(), cid, application.AddWithholdingRequest{
		PurchaseID: id, ConceptID: body.ConceptID, Base: body.Base,
	})
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPurchaseNotConfirmed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toWithholdingDTO(*wh))
}

func (h *Handler) handleListWithholdings(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	list, err := h.withholding.List(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]withholdingDTO, len(list))
	for i, wh := range list {
		dtos[i] = toWithholdingDTO(wh)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleGetPDF(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	pdfBytes, err := h.pdf.GetPDF(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="orden-compra.pdf"`)
	_, _ = w.Write(pdfBytes)
}

type sendPurchaseEmailBody struct {
	To string `json:"to,omitempty"`
}

func (h *Handler) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body sendPurchaseEmailBody
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, http.StatusBadRequest, "cuerpo inválido")
			return
		}
	}
	if err := h.sendEmail.Send(r.Context(), cid, id, body.To); err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (h *Handler) handleRecordPayment(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.RecordPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.payment.Record(r.Context(), cid, req)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toPaymentDTO(p))
}

func (h *Handler) handleListPayments(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	purchaseID, err := uuid.Parse(r.PathValue("purchase_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "purchase_id inválido")
		return
	}
	list, err := h.payment.ListByPurchase(r.Context(), cid, purchaseID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]paymentDTO, len(list))
	for i, p := range list {
		dtos[i] = toPaymentDTO(&p)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleGetPayables(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.payment.GetPayables(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]payableDTO, len(list))
	for i, b := range list {
		dtos[i] = toPayableDTO(b)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	o, err := h.create.Execute(r.Context(), cid, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, toPurchaseDTO(o))
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.get.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]purchaseDTO, len(list))
	for i, o := range list {
		dtos[i] = toPurchaseDTO(&o)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	o, err := h.get.ByID(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toPurchaseDTO(o))
}

func (h *Handler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	o, err := h.confirm.Execute(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPurchaseNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "purchase.confirmed", "purchase", o.ID, map[string]any{"number": o.Number})
	respond(w, http.StatusOK, toPurchaseDTO(o))
}

func (h *Handler) handleReceive(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	o, err := h.receive.Execute(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPurchaseNotConfirmed) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "purchase.received", "purchase", o.ID, map[string]any{"number": o.Number})
	respond(w, http.StatusOK, toPurchaseDTO(o))
}

func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.cancel.Execute(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.delete.Execute(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrPurchaseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrPurchaseNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
