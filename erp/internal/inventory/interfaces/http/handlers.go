package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/inventory/application"
	"github.com/diegofxm/erp/internal/inventory/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	move *application.MoveUseCase
	get  *application.GetUseCase
}

func NewHandler(move *application.MoveUseCase, get *application.GetUseCase) *Handler {
	return &Handler{move: move, get: get}
}

func (h *Handler) handleMove(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	m, err := h.move.Execute(r.Context(), companyID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, m)
}

func (h *Handler) handleListStock(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.get.Stock(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.StockEntry{}
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) handleListMovements(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var productID *uuid.UUID
	if raw := r.URL.Query().Get("product_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "product_id inválido")
			return
		}
		productID = &id
	}
	list, err := h.get.Movements(r.Context(), companyID, productID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Movement{}
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
