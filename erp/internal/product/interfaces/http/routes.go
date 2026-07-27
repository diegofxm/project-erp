package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/products", h.handleList)
	mux.HandleFunc("POST /api/v1/products", h.handleCreate)
	mux.HandleFunc("GET /api/v1/products/{id}", h.handleGetByID)
	mux.HandleFunc("PUT /api/v1/products/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE /api/v1/products/{id}", h.handleDelete)
}
