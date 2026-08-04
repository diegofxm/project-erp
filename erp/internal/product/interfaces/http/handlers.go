package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/product/application"
	"github.com/diegofxm/erp/internal/product/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

type Handler struct {
	create *application.CreateUseCase
	get    *application.GetUseCase
	update *application.UpdateUseCase
	delete *application.DeleteUseCase
	audit  AuditLogger
}

func NewHandler(
	create *application.CreateUseCase,
	get *application.GetUseCase,
	update *application.UpdateUseCase,
	del *application.DeleteUseCase,
	audit AuditLogger,
) *Handler {
	return &Handler{create: create, get: get, update: update, delete: del, audit: audit}
}

func (h *Handler) logProduct(ctx context.Context, companyID uuid.UUID, action string, p *domain.Product) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	h.audit.Log(ctx, companyID, userID, "product."+action, "product", &p.ID, map[string]any{
		"code": p.Code, "name": p.Name,
	})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.create.Execute(r.Context(), companyID, req)
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateProduct) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logProduct(r.Context(), companyID, "created", p)
	respond(w, http.StatusCreated, p)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.get.List(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Product{}
	}
	respond(w, http.StatusOK, map[string]any{"products": list, "count": len(list)})
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.get.ByID(r.Context(), companyID, id)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, p)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req application.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.update.Execute(r.Context(), companyID, id, req)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logProduct(r.Context(), companyID, "updated", p)
	respond(w, http.StatusOK, p)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.delete.Execute(r.Context(), companyID, id); err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "product.deleted", "product", &id, nil)
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
