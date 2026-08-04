package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/tenant"
	"github.com/diegofxm/erp/internal/thirdparty/application"
	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// Handler expone un catálogo (Clientes o Proveedores) sobre el módulo unificado de terceros —
// role fija cuál. Las rutas, el wrapper del listado ("customers"/"suppliers") y la forma del
// JSON quedan iguales a los antiguos módulos customer/ y supplier/, así el frontend no necesitó
// ningún cambio al migrar.
type Handler struct {
	create *application.CreateUseCase
	get    *application.GetUseCase
	update *application.UpdateUseCase
	delete *application.DeleteUseCase
	audit  AuditLogger

	role        domain.Role
	pathPrefix  string
	listKey     string
	notFoundErr error
	dupErr      error
}

func NewCustomerHandler(create *application.CreateUseCase, get *application.GetUseCase, update *application.UpdateUseCase, del *application.DeleteUseCase, audit AuditLogger) *Handler {
	return &Handler{
		create: create, get: get, update: update, delete: del, audit: audit,
		role: domain.RoleCustomer, pathPrefix: "/api/v1/customers", listKey: "customers",
		notFoundErr: domain.ErrCustomerNotFound, dupErr: domain.ErrDuplicateCustomer,
	}
}

func NewSupplierHandler(create *application.CreateUseCase, get *application.GetUseCase, update *application.UpdateUseCase, del *application.DeleteUseCase, audit AuditLogger) *Handler {
	return &Handler{
		create: create, get: get, update: update, delete: del, audit: audit,
		role: domain.RoleSupplier, pathPrefix: "/api/v1/suppliers", listKey: "suppliers",
		notFoundErr: domain.ErrSupplierNotFound, dupErr: domain.ErrDuplicateSupplier,
	}
}

// logParty registra un evento de auditoría sobre un tercero, con el rol (customer/supplier)
// como prefijo de la acción (ej. "customer.created", "supplier.updated").
func (h *Handler) logParty(ctx context.Context, companyID uuid.UUID, action string, p *domain.Party) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	h.audit.Log(ctx, companyID, userID, string(h.role)+"."+action, "party", &p.ID, map[string]any{
		"name":                    p.Name,
		"identification_number":   p.IdentificationNumber,
	})
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+h.pathPrefix, h.handleList)
	mux.HandleFunc("POST "+h.pathPrefix, h.handleCreate)
	mux.HandleFunc("GET "+h.pathPrefix+"/{id}", h.handleGetByID)
	mux.HandleFunc("PUT "+h.pathPrefix+"/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE "+h.pathPrefix+"/{id}", h.handleDelete)
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

	p, err := h.create.Execute(r.Context(), companyID, h.role, req)
	if err != nil {
		if errors.Is(err, h.dupErr) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logParty(r.Context(), companyID, "created", p)
	respond(w, http.StatusCreated, p)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	list, err := h.get.List(r.Context(), companyID, h.role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Party{}
	}
	respond(w, http.StatusOK, map[string]any{h.listKey: list, "count": len(list)})
}

func (h *Handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	p, err := h.get.ByID(r.Context(), companyID, id, h.role)
	if err != nil {
		if errors.Is(err, h.notFoundErr) {
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

	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	var req application.SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	p, err := h.update.Execute(r.Context(), companyID, id, h.role, req)
	if err != nil {
		if errors.Is(err, h.notFoundErr) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.logParty(r.Context(), companyID, "updated", p)
	respond(w, http.StatusOK, p)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	if err := h.delete.Execute(r.Context(), companyID, id, h.role); err != nil {
		if errors.Is(err, h.notFoundErr) {
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
		h.audit.Log(r.Context(), companyID, userID, string(h.role)+".role_removed", "party", &id, nil)
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(r.PathValue(param))
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
