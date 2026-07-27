package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/hr/absences", h.handleListAbsences)
	mux.HandleFunc("POST /api/v1/hr/absences", h.handleRequestAbsence)
	mux.HandleFunc("GET /api/v1/hr/absences/{id}", h.handleGetAbsence)
	mux.HandleFunc("POST /api/v1/hr/absences/{id}/approve", h.handleApproveAbsence)
	mux.HandleFunc("POST /api/v1/hr/absences/{id}/reject", h.handleRejectAbsence)
}
