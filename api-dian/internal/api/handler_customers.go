package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/api-dian/internal/api/middleware"
	"github.com/diegofxm/api-dian/internal/customers"
	"github.com/google/uuid"

	"github.com/diegofxm/api-dian/internal/api/response"
)

// customerResponse expone el mismo contrato partyDTO que ya usan invoices/credit-notes/
// debit-notes para Customer — así el frontend puede tomar la respuesta de
// GET /customers/{id} y mandarla tal cual como "customer" al emitir un documento.
type customerResponse struct {
	ID uuid.UUID `json:"id"`
	partyDTO
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func customerToResponse(c *customers.Customer) customerResponse {
	return customerResponse{
		ID:        c.ID,
		partyDTO:  partyFromDomain(c.Party),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (a *API) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req partyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	c, err := a.customers.CreateCustomer(r.Context(), middleware.GetTenantID(r.Context()), req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, customerToResponse(c))
}

// handleGetCustomer exige que el cliente pertenezca al emisor autenticado — si no, el mismo
// 404 que un cliente inexistente (customers.ErrCustomerNotFound).
func (a *API) handleGetCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	c, err := a.customers.GetCustomer(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if c.IssuerID != middleware.GetTenantID(r.Context()) {
		response.WriteError(w, customers.ErrCustomerNotFound)
		return
	}

	response.WriteJSON(w, http.StatusOK, customerToResponse(c))
}

func (a *API) handleListCustomers(w http.ResponseWriter, r *http.Request) {
	list, err := a.customers.ListCustomers(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]customerResponse, len(list))
	for i, c := range list {
		out[i] = customerToResponse(c)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"customers": out, "count": len(out)})
}

func (a *API) handleUpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var req partyDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	c, err := a.customers.UpdateCustomer(r.Context(), middleware.GetTenantID(r.Context()), id, req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, customerToResponse(c))
}

func (a *API) handleDeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := a.customers.DeleteCustomer(r.Context(), middleware.GetTenantID(r.Context()), id); err != nil {
		response.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
