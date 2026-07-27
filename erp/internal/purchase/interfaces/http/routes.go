package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/purchases", h.handleList)
	mux.HandleFunc("POST /api/v1/purchases", h.handleCreate)
	mux.HandleFunc("GET /api/v1/purchases/{id}", h.handleGetByID)
	mux.HandleFunc("POST /api/v1/purchases/{id}/confirm", h.handleConfirm)
	mux.HandleFunc("POST /api/v1/purchases/{id}/cancel", h.handleCancel)
}
