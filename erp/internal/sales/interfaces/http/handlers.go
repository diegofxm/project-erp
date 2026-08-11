package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/erp/internal/sales/application"
	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// ── DTOs de salida ──────────────────────────────────────────────────────────────────────────
// domain.Sale/Quote/SalePayment/ReceivableBalance no llevan tags json (no deben — el dominio no
// conoce HTTP) — acá se mapean a snake_case, igual que el resto de los módulos (ver
// electronic/interfaces/http toNumberingRangeDTO).

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

type saleLineDTO struct {
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

// paymentMeanDTO -- mismo formato snake_case que electronic/interfaces/http para el mismo tipo
// cofdom.PaymentMean, así el frontend reutiliza PaymentMeansEditor.tsx sin adaptar nada.
type paymentMeanDTO struct {
	Code              string `json:"code"`
	PaymentMethodCode string `json:"payment_method_code"`
	DueDate           string `json:"due_date,omitempty"`
	PaymentReference  string `json:"payment_reference,omitempty"`
}

func toPaymentMeanDTOs(pms []cofdom.PaymentMean) []paymentMeanDTO {
	out := make([]paymentMeanDTO, len(pms))
	for i, pm := range pms {
		out[i] = paymentMeanDTO{
			Code: pm.Code, PaymentMethodCode: pm.PaymentMethodCode,
			DueDate: pm.DueDate, PaymentReference: pm.PaymentReference,
		}
	}
	return out
}

type saleDTO struct {
	ID                uuid.UUID        `json:"id"`
	CompanyID         uuid.UUID        `json:"company_id"`
	CustomerID        uuid.UUID        `json:"customer_id"`
	Number            string           `json:"number"`
	Status            string           `json:"status"`
	IssueDate         string           `json:"issue_date"`
	DueDate           string           `json:"due_date,omitempty"`
	Notes             string           `json:"notes"`
	Lines             []saleLineDTO    `json:"lines"`
	PaymentMeans      []paymentMeanDTO `json:"payment_means"`
	InvoiceDocumentID *uuid.UUID       `json:"invoice_document_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

func toSaleDTO(s *domain.Sale) saleDTO {
	lines := make([]saleLineDTO, len(s.Lines))
	for i, l := range s.Lines {
		lines[i] = saleLineDTO{
			ID: l.ID, ProductID: l.ProductID, Description: l.Description,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, Discount: l.Discount, TaxRate: l.TaxRate,
			Subtotal: l.Subtotal, TaxAmount: l.TaxAmount, Total: l.Total,
		}
	}
	return saleDTO{
		ID: s.ID, CompanyID: s.CompanyID, CustomerID: s.CustomerID,
		Number: s.Number, Status: string(s.Status),
		IssueDate: formatDate(s.IssueDate), DueDate: formatDatePtr(s.DueDate),
		Notes: s.Notes, Lines: lines, PaymentMeans: toPaymentMeanDTOs(s.PaymentMeans),
		InvoiceDocumentID: s.InvoiceDocumentID,
		CreatedAt:         s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

type quoteLineDTO struct {
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

type quoteDTO struct {
	ID           uuid.UUID        `json:"id"`
	CompanyID    uuid.UUID        `json:"company_id"`
	CustomerID   uuid.UUID        `json:"customer_id"`
	Number       string           `json:"number"`
	Status       string           `json:"status"`
	IssueDate    string           `json:"issue_date"`
	ValidUntil   string           `json:"valid_until,omitempty"`
	Notes        string           `json:"notes"`
	Lines        []quoteLineDTO   `json:"lines"`
	PaymentMeans []paymentMeanDTO `json:"payment_means"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

func toQuoteDTO(q *domain.Quote) quoteDTO {
	lines := make([]quoteLineDTO, len(q.Lines))
	for i, l := range q.Lines {
		lines[i] = quoteLineDTO{
			ID: l.ID, ProductID: l.ProductID, Description: l.Description,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, Discount: l.Discount, TaxRate: l.TaxRate,
			Subtotal: l.Subtotal, TaxAmount: l.TaxAmount, Total: l.Total,
		}
	}
	return quoteDTO{
		ID: q.ID, CompanyID: q.CompanyID, CustomerID: q.CustomerID,
		Number: q.Number, Status: string(q.Status),
		IssueDate: formatDate(q.IssueDate), ValidUntil: formatDatePtr(q.ValidUntil),
		Notes: q.Notes, Lines: lines, PaymentMeans: toPaymentMeanDTOs(q.PaymentMeans),
		CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}

type paymentDTO struct {
	ID            uuid.UUID `json:"id"`
	CompanyID     uuid.UUID `json:"company_id"`
	SaleID        uuid.UUID `json:"sale_id"`
	PaymentDate   string    `json:"payment_date"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Reference     string    `json:"reference"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPaymentDTO(p *domain.SalePayment) paymentDTO {
	return paymentDTO{
		ID: p.ID, CompanyID: p.CompanyID, SaleID: p.SaleID,
		PaymentDate: formatDate(p.PaymentDate), Amount: p.Amount,
		PaymentMethod: string(p.PaymentMethod), Reference: p.Reference, Notes: p.Notes,
		CreatedAt: p.CreatedAt,
	}
}

type receivableDTO struct {
	SaleID     uuid.UUID `json:"sale_id"`
	SaleNumber string    `json:"sale_number"`
	CustomerID uuid.UUID `json:"customer_id"`
	IssueDate  string    `json:"issue_date"`
	DueDate    string    `json:"due_date,omitempty"`
	Total      float64   `json:"total"`
	Paid       float64   `json:"paid"`
	Balance    float64   `json:"balance"`
}

func toReceivableDTO(r domain.ReceivableBalance) receivableDTO {
	return receivableDTO{
		SaleID: r.SaleID, SaleNumber: r.SaleNumber, CustomerID: r.CustomerID,
		IssueDate: formatDate(r.IssueDate), DueDate: formatDatePtr(r.DueDate),
		Total: r.Total, Paid: r.Paid, Balance: r.Balance,
	}
}

type Handler struct {
	create         *application.CreateUseCase
	update         *application.UpdateUseCase
	get            *application.GetUseCase
	confirm        *application.ConfirmUseCase
	cancel         *application.CancelUseCase
	quote          *application.QuoteUseCase
	payment        *application.PaymentUseCase
	quotePDF       *application.GetQuotePDFUseCase
	sendQuoteEmail *application.SendQuoteEmailUseCase
	setCounter     *application.SetNumberCounterUseCase
	delete         *application.DeleteUseCase
	audit          AuditLogger
}

func NewHandler(
	create *application.CreateUseCase,
	update *application.UpdateUseCase,
	get *application.GetUseCase,
	confirm *application.ConfirmUseCase,
	cancel *application.CancelUseCase,
	quote *application.QuoteUseCase,
	payment *application.PaymentUseCase,
	quotePDF *application.GetQuotePDFUseCase,
	sendQuoteEmail *application.SendQuoteEmailUseCase,
	setCounter *application.SetNumberCounterUseCase,
	del *application.DeleteUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{
		create: create, update: update, get: get, confirm: confirm, cancel: cancel, quote: quote, payment: payment,
		quotePDF: quotePDF, sendQuoteEmail: sendQuoteEmail, setCounter: setCounter, delete: del, audit: audit,
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
	s, err := h.create.Execute(r.Context(), cid, req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, toSaleDTO(s))
}

// handleUpdate corrige cliente/fecha/notas/líneas de una venta EN BORRADOR -- el repositorio
// rechaza (ErrSaleNotDraft) si ya está confirmada.
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req application.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	s, err := h.update.Execute(r.Context(), cid, id, req)
	if err != nil {
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrSaleNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "sale.updated", "sale", s.ID, nil)
	respond(w, http.StatusOK, toSaleDTO(s))
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
	dtos := make([]saleDTO, len(list))
	for i := range list {
		dtos[i] = toSaleDTO(&list[i])
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
	s, err := h.get.ByID(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toSaleDTO(s))
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
	s, err := h.confirm.Execute(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrSaleNotDraft) ||
			errors.Is(err, domain.ErrInsufficientStock) ||
			errors.Is(err, domain.ErrCreditLimitExceeded) ||
			errors.Is(err, domain.ErrOverdueBalance) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "sale.confirmed", "sale", s.ID, map[string]any{"number": s.Number})
	respond(w, http.StatusOK, toSaleDTO(s))
}

// handleCancel cancela una venta ya confirmada -- reversa efectos contables/inventario reales,
// por eso requiere rol de administración (ver plan de acción 2026-08-09, Fase 2 punto 08:
// decisión explícita del responsable del proyecto, distinta de recepción/pago en purchase que
// sí queda abierta a cualquier member).
func (h *Handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireManage(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.cancel.Execute(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "sale.cancelled", "sale", id, nil)
	respond(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ── Cotizaciones ──────────────────────────────────────────────────────────────────────────

func (h *Handler) handleCreateQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	q, err := h.quote.Create(r.Context(), cid, req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusCreated, toQuoteDTO(q))
}

func (h *Handler) handleUpdateQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req application.CreateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	q, err := h.quote.Update(r.Context(), cid, id, req)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrQuoteNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respond(w, http.StatusOK, toQuoteDTO(q))
}

func (h *Handler) handleListQuotes(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.quote.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]quoteDTO, len(list))
	for i := range list {
		dtos[i] = toQuoteDTO(&list[i])
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleGetQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	q, err := h.quote.GetByID(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, toQuoteDTO(q))
}

func (h *Handler) handleSendQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	q, err := h.quote.Send(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, toQuoteDTO(q))
}

func (h *Handler) handleGetQuotePDF(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	pdfBytes, err := h.quotePDF.GetPDF(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="cotizacion.pdf"`)
	_, _ = w.Write(pdfBytes)
}

type sendQuoteEmailBody struct {
	To string `json:"to,omitempty"`
}

func (h *Handler) handleSendQuoteEmail(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body sendQuoteEmailBody
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, http.StatusBadRequest, "cuerpo inválido")
			return
		}
	}
	q, err := h.sendQuoteEmail.Send(r.Context(), cid, id, body.To)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, toQuoteDTO(q))
}

func (h *Handler) handleAcceptQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	q, err := h.quote.Accept(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, toQuoteDTO(q))
}

func (h *Handler) handleRejectQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.quote.Reject(r.Context(), cid, id); err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *Handler) handleConvertQuoteToSale(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	sale, err := h.quote.ConvertToSale(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respond(w, http.StatusCreated, toSaleDTO(sale))
}

// handleDeleteSale elimina una venta en borrador -- Repository.Delete ya valida el estado.
func (h *Handler) handleDeleteSale(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrSaleNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.audit != nil {
		h.logAudit(r.Context(), cid, "sale.deleted", "sale", id, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDeleteQuote(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.quote.Delete(r.Context(), cid, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Pagos / Cartera ───────────────────────────────────────────────────────────────────────

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
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.logAudit(r.Context(), cid, "sale.payment_recorded", "sale_payment", p.ID, map[string]any{"sale_id": p.SaleID, "amount": p.Amount})
	respond(w, http.StatusCreated, toPaymentDTO(p))
}

func (h *Handler) handleListPayments(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(r.PathValue("sale_id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "sale_id inválido")
		return
	}
	list, err := h.payment.ListBySale(r.Context(), cid, saleID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]paymentDTO, len(list))
	for i := range list {
		dtos[i] = toPaymentDTO(&list[i])
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleGetReceivables(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.payment.GetReceivables(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]receivableDTO, len(list))
	for i, rb := range list {
		dtos[i] = toReceivableDTO(rb)
	}
	respond(w, http.StatusOK, dtos)
}

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func requireManage(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !tenant.CanManage(r.Context()) {
		respondError(w, http.StatusForbidden, "requiere rol de administrador")
		return uuid.Nil, false
	}
	return cid, true
}

// ── Consecutivos ──────────────────────────────────────────────────────────────────────────────

type setNumberCounterBody struct {
	DocType    string `json:"doc_type"` // "sale" | "quote"
	Year       int    `json:"year"`
	NextNumber int    `json:"next_number"`
}

// handleSetNumberCounter fija el próximo consecutivo de venta o cotización — pensado para migrar
// una empresa que ya traía su propia numeración de otro sistema. Solo admin/owner.
func (h *Handler) handleSetNumberCounter(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireManage(w, r)
	if !ok {
		return
	}
	var body setNumberCounterBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	var (
		seq int
		err error
	)
	switch body.DocType {
	case "sale":
		seq, err = h.setCounter.SetSale(r.Context(), cid, body.Year, body.NextNumber)
	case "quote":
		seq, err = h.setCounter.SetQuote(r.Context(), cid, body.Year, body.NextNumber)
	default:
		respondError(w, http.StatusBadRequest, `doc_type debe ser "sale" o "quote"`)
		return
	}
	if err != nil {
		if errors.Is(err, domain.ErrNumberCounterInvalid) || errors.Is(err, domain.ErrNumberCounterBackwards) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"doc_type": body.DocType, "year": body.Year, "last_seq": seq, "next_will_be": seq + 1,
	})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
