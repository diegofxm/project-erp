package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/tenant"
	"github.com/diegofxm/erp/internal/thirdparty/application"
	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

// Handler expone un catálogo (Clientes o Proveedores) sobre el módulo unificado de terceros —
// role fija cuál. Las rutas, el wrapper del listado ("customers"/"suppliers") y la forma del
// JSON quedan iguales a los antiguos módulos customer/ y supplier/, así el frontend no necesitó
// ningún cambio al migrar.
type Handler struct {
	create *application.CreateUseCase
	get    *application.GetUseCase
	update *application.UpdateUseCase
	delete *application.DeleteUseCase

	role        domain.Role
	pathPrefix  string
	listKey     string
	notFoundErr error
	dupErr      error
}

func NewCustomerHandler(create *application.CreateUseCase, get *application.GetUseCase, update *application.UpdateUseCase, del *application.DeleteUseCase) *Handler {
	return &Handler{
		create: create, get: get, update: update, delete: del,
		role: domain.RoleCustomer, pathPrefix: "/api/v1/customers", listKey: "customers",
		notFoundErr: domain.ErrCustomerNotFound, dupErr: domain.ErrDuplicateCustomer,
	}
}

func NewSupplierHandler(create *application.CreateUseCase, get *application.GetUseCase, update *application.UpdateUseCase, del *application.DeleteUseCase) *Handler {
	return &Handler{
		create: create, get: get, update: update, delete: del,
		role: domain.RoleSupplier, pathPrefix: "/api/v1/suppliers", listKey: "suppliers",
		notFoundErr: domain.ErrSupplierNotFound, dupErr: domain.ErrDuplicateSupplier,
	}
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
