package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sales", h.handleList)
	mux.HandleFunc("POST /api/v1/sales", h.handleCreate)
	mux.HandleFunc("GET /api/v1/sales/{id}", h.handleGetByID)
	mux.HandleFunc("POST /api/v1/sales/{id}/confirm", h.handleConfirm)
	mux.HandleFunc("POST /api/v1/sales/{id}/cancel", h.handleCancel)
}
