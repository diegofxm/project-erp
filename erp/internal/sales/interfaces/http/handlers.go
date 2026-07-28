package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/application"
	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	create  *application.CreateUseCase
	get     *application.GetUseCase
	confirm *application.ConfirmUseCase
	cancel  *application.CancelUseCase
	quote   *application.QuoteUseCase
	payment *application.PaymentUseCase
}

func NewHandler(
	create *application.CreateUseCase,
	get *application.GetUseCase,
	confirm *application.ConfirmUseCase,
	cancel *application.CancelUseCase,
	quote *application.QuoteUseCase,
	payment *application.PaymentUseCase,
) *Handler {
	return &Handler{create: create, get: get, confirm: confirm, cancel: cancel, quote: quote, payment: payment}
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
	respond(w, http.StatusCreated, s)
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
	if list == nil {
		list = []domain.Sale{}
	}
	respond(w, http.StatusOK, list)
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
	respond(w, http.StatusOK, s)
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
		if errors.Is(err, domain.ErrSaleNotDraft) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, s)
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
		if errors.Is(err, domain.ErrSaleNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
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
	respond(w, http.StatusCreated, q)
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
	if list == nil {
		list = []domain.Quote{}
	}
	respond(w, http.StatusOK, list)
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
	respond(w, http.StatusOK, q)
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
	respond(w, http.StatusOK, q)
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
	respond(w, http.StatusOK, q)
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
	respond(w, http.StatusCreated, sale)
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
	respond(w, http.StatusCreated, p)
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
	if list == nil {
		list = []domain.SalePayment{}
	}
	respond(w, http.StatusOK, list)
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
	if list == nil {
		list = []domain.ReceivableBalance{}
	}
	respond(w, http.StatusOK, list)
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
