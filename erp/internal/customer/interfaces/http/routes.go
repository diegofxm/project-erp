package http

import "net/http"

// RegisterRoutes monta las rutas del módulo customer.
//
//   GET    /api/v1/customers           — listar clientes de la empresa activa
//   POST   /api/v1/customers           — crear cliente
//   GET    /api/v1/customers/{id}      — obtener cliente por ID
//   PUT    /api/v1/customers/{id}      — actualizar cliente
//   DELETE /api/v1/customers/{id}      — eliminar cliente
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers", h.handleList)
	mux.HandleFunc("POST /api/v1/customers", h.handleCreate)
	mux.HandleFunc("GET /api/v1/customers/{id}", h.handleGetByID)
	mux.HandleFunc("PUT /api/v1/customers/{id}", h.handleUpdate)
	mux.HandleFunc("DELETE /api/v1/customers/{id}", h.handleDelete)
}
