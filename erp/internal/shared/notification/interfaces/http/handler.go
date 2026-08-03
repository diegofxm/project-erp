package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/notification/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type useCase interface {
	Execute(ctx context.Context, companyID uuid.UUID) ([]domain.Alert, error)
}

type Handler struct {
	uc useCase
}

func NewHandler(uc useCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/notifications/system-alerts", h.handleListSystemAlerts)
}

func (h *Handler) handleListSystemAlerts(w http.ResponseWriter, r *http.Request) {
	companyID := tenant.GetCompanyID(r.Context())
	if companyID == uuid.Nil {
		http.Error(w, `{"error":"empresa activa requerida"}`, http.StatusUnauthorized)
		return
	}

	alerts, err := h.uc.Execute(r.Context(), companyID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if alerts == nil {
		alerts = []domain.Alert{}
	}

	out := make([]map[string]any, len(alerts))
	for i, a := range alerts {
		out[i] = map[string]any{
			"id":       a.ID,
			"severity": a.Severity,
			"title":    a.Title,
			"message":  a.Message,
			"link_to":  a.LinkTo,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"alerts": out,
		"count":  len(out),
	})
}
