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
}

func NewHandler(
	create *application.CreateUseCase,
	get *application.GetUseCase,
	confirm *application.ConfirmUseCase,
	cancel *application.CancelUseCase,
) *Handler {
	return &Handler{create: create, get: get, confirm: confirm, cancel: cancel}
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
