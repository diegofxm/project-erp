package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/suppliers", h.handleList)
	mux.HandleFunc("POST /api/v1/suppliers", h.handleCreate)
	mux.HandleFunc("GET /api/v1/suppliers/{id}", h.handleGetByID)
	mux.HandleFunc("PUT /api/v1/suppliers/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE /api/v1/suppliers/{id}", h.handleDelete)
}
