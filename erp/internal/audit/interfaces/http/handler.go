package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/audit/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type useCase interface {
	List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Event, error)
}

type Handler struct {
	uc useCase
}

func NewHandler(uc useCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/audit-events", h.handleList)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	companyID := tenant.GetCompanyID(r.Context())
	if companyID == uuid.Nil {
		http.Error(w, `{"error":"empresa activa requerida"}`, http.StatusUnauthorized)
		return
	}

	filter := domain.ListFilter{Limit: 200}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	if v := r.URL.Query().Get("resource_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ResourceID = &id
		}
	}

	events, err := h.uc.List(r.Context(), companyID, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if events == nil {
		events = []*domain.Event{}
	}

	out := make([]map[string]any, len(events))
	for i, e := range events {
		out[i] = map[string]any{
			"id":            e.ID,
			"action":        e.Action,
			"resource_type": e.ResourceType,
			"resource_id":   e.ResourceID,
			"metadata":      e.Metadata,
			"user_email":    e.UserEmail,
			"user_name":     e.UserName,
			"created_at":    e.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"events": out,
		"count":  len(out),
	})
}
