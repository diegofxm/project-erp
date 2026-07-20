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

func (a *API) handleAdminUpdateIssuerSettings(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	a.patchSettings(w, r, issuerID)
}

// patchSettings es la lógica compartida de PATCH settings — acepta brand_color, tarifas y precio por documento.
// Las fechas de afiliación/renovación se gestionan por los endpoints /affiliate y /renew.
func (a *API) patchSettings(w http.ResponseWriter, r *http.Request, issuerID uuid.UUID) {
	var body struct {
		BrandColor          string `json:"brand_color"`
		PricePerDocumentCOP *int   `json:"price_per_document_cop"`
		AffiliationFeeCOP   *int   `json:"affiliation_fee_cop"`
		RenewalFeeCOP       *int   `json:"renewal_fee_cop"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	current, err := a.settings.Get(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	color := current.BrandColor
	if body.BrandColor != "" {
		color = body.BrandColor
	}
	price := current.PricePerDocumentCOP
	if body.PricePerDocumentCOP != nil {
		price = *body.PricePerDocumentCOP
	}
	affFee := current.AffiliationFeeCOP
	if body.AffiliationFeeCOP != nil {
		affFee = *body.AffiliationFeeCOP
	}
	renFee := current.RenewalFeeCOP
	if body.RenewalFeeCOP != nil {
		renFee = *body.RenewalFeeCOP
	}

	st, err := a.settings.Update(r.Context(), issuerID, color, price, affFee, renFee)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, st)
}
