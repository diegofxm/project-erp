package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/tenant"
	"github.com/diegofxm/erp/internal/stats/application"
)

type Handler struct {
	billing *application.GetBillingStatsUseCase
}

func NewHandler(billing *application.GetBillingStatsUseCase) *Handler {
	return &Handler{billing: billing}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/stats/billing", h.handleBilling)
}

func (h *Handler) handleBilling(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	stats, err := h.billing.Execute(r.Context(), companyID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func mustTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		writeErr(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
