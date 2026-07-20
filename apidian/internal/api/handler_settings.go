package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
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

	var body struct {
		BrandColor string `json:"brand_color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	st, err := a.settings.UpdateBrandColor(r.Context(), issuerID, body.BrandColor)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, st)
}
