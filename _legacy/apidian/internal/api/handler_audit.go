package api

import (
	"net/http"
	"strconv"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/audit"
	"github.com/google/uuid"
)

// auditEventResponse es la representación pública de un evento de auditoría.
type auditEventResponse struct {
	ID           uuid.UUID      `json:"id"`
	UserName     string         `json:"user_name,omitempty"`
	UserEmail    string         `json:"user_email,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID     `json:"resource_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    string         `json:"created_at"`
}

func auditEventToResponse(e audit.Event) auditEventResponse {
	return auditEventResponse{
		ID:           e.ID,
		UserName:     e.UserName,
		UserEmail:    e.UserEmail,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		Metadata:     e.Metadata,
		CreatedAt:    e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// handleListAuditEvents devuelve el historial de actividad del emisor autenticado.
// Acepta ?resource_id=<uuid> para filtrar por documento, ?limit=&offset= para paginar.
func (a *API) handleListAuditEvents(w http.ResponseWriter, r *http.Request) {
	if a.auditSvc == nil {
		response.WriteJSON(w, http.StatusOK, map[string]any{"events": []any{}, "count": 0})
		return
	}

	q := r.URL.Query()
	filter := audit.ListFilter{}

	if s := q.Get("resource_id"); s != "" {
		id, ok := parseUUID(w, s)
		if !ok {
			return
		}
		filter.ResourceID = &id
	}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "limit inválido"})
			return
		}
		filter.Limit = n
	}
	if s := q.Get("offset"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "offset inválido"})
			return
		}
		filter.Offset = n
	}

	events, err := a.auditSvc.List(r.Context(), middleware.GetTenantID(r.Context()), filter)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]auditEventResponse, len(events))
	for i, e := range events {
		out[i] = auditEventToResponse(e)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"events": out, "count": len(out)})
}

// logEvent registra un evento de auditoría de forma fire-and-forget (no propaga errores).
// Solo actúa si a.auditSvc != nil (nil en tests con repositorios en memoria).
func (a *API) logEvent(r *http.Request, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any) {
	if a.auditSvc == nil {
		return
	}
	userID := middleware.GetUserID(r.Context())
	issuerID := middleware.GetTenantID(r.Context())
	e := audit.Event{
		IssuerID:     issuerID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
	}
	if userID != (uuid.UUID{}) {
		e.UserID = &userID
	}
	a.auditSvc.Log(r.Context(), e)
}
