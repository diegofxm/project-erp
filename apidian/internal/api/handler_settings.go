package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/google/uuid"
)

func (a *API) handleGetMySettings(w http.ResponseWriter, r *http.Request) {
	issuerID := middleware.GetTenantID(r.Context())
	st, err := a.settings.Get(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, st)
}

func (a *API) handleUpdateMySettings(w http.ResponseWriter, r *http.Request) {
	issuerID := middleware.GetTenantID(r.Context())
	a.patchSettings(w, r, issuerID)
}

// handleAdminGetIssuerSettings permite al superadmin leer la configuración de cualquier emisor.
func (a *API) handleAdminGetIssuerSettings(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	st, err := a.settings.Get(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, st)
}

// handleAdminUpdateIssuerSettings permite al superadmin actualizar la configuración de cualquier emisor.
func (a *API) handleAdminUpdateIssuerSettings(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	a.patchSettings(w, r, issuerID)
}

// patchSettings es la lógica compartida de PATCH settings — acepta brand_color y/o price_per_document_cop.
func (a *API) patchSettings(w http.ResponseWriter, r *http.Request, issuerID uuid.UUID) {
	var body struct {
		BrandColor          string `json:"brand_color"`
		PricePerDocumentCOP *int   `json:"price_per_document_cop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	// Leer estado actual para no pisar campos no enviados.
	current, err := a.settings.Get(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	price := current.PricePerDocumentCOP
	if body.PricePerDocumentCOP != nil {
		price = *body.PricePerDocumentCOP
	}
	color := current.BrandColor
	if body.BrandColor != "" {
		color = body.BrandColor
	}

	st, err := a.settings.Update(r.Context(), issuerID, color, price)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, st)
}
