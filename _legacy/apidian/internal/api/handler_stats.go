package api

import (
	"net/http"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
)

// handleGetBillingStats devuelve las métricas de facturación del emisor activo.
// Todas las métricas están acotadas al issuer del token — nunca cruza datos entre tenants.
func (a *API) handleGetBillingStats(w http.ResponseWriter, r *http.Request) {
	issuerID := middleware.GetTenantID(r.Context())
	stats, err := a.documents.GetBillingStats(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, stats)
}
