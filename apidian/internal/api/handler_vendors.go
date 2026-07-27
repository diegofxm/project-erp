package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/edocuments/vendors"
	"github.com/google/uuid"
)

// vendorResponse expone el mismo contrato partyDTO que customerResponse — el frontend puede
// tomar la respuesta de GET /vendors/{id} y mandarla tal cual como "vendor" al emitir un DS.
type vendorResponse struct {
	ID uuid.UUID `json:"id"`
	partyDTO
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func vendorToResponse(v *vendors.Vendor) vendorResponse {
	return vendorResponse{
		ID:        v.ID,
		partyDTO:  partyFromDomain(v.Party),
		CreatedAt: v.CreatedAt,
		UpdatedAt: v.UpdatedAt,
	}
}

func (a *API) handleCreateVendor(w http.ResponseWriter, r *http.Request) {
	var req partyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	v, err := a.vendors.CreateVendor(r.Context(), middleware.GetTenantID(r.Context()), req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, vendorToResponse(v))
}

func (a *API) handleGetVendor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	v, err := a.vendors.GetVendor(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if v.IssuerID != middleware.GetTenantID(r.Context()) {
		response.WriteError(w, vendors.ErrVendorNotFound)
		return
	}

	response.WriteJSON(w, http.StatusOK, vendorToResponse(v))
}

func (a *API) handleListVendors(w http.ResponseWriter, r *http.Request) {
	list, err := a.vendors.ListVendors(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]vendorResponse, len(list))
	for i, v := range list {
		out[i] = vendorToResponse(v)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"vendors": out, "count": len(out)})
}

func (a *API) handleUpdateVendor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var req partyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	v, err := a.vendors.UpdateVendor(r.Context(), middleware.GetTenantID(r.Context()), id, req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, vendorToResponse(v))
}

func (a *API) handleDeleteVendor(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := a.vendors.DeleteVendor(r.Context(), middleware.GetTenantID(r.Context()), id); err != nil {
		response.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
