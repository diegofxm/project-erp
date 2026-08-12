package http

import "net/http"

// RegisterRoutes monta las rutas del módulo inventory.
//
//   GET    /api/v1/inventory/stock            — stock actual de todos los productos
//   GET    /api/v1/inventory/movements        — historial (?product_id=uuid filtra por producto)
//   POST   /api/v1/inventory/movements        — registrar movimiento (entry/exit/transfer/adjust)
//   DELETE /api/v1/inventory/movements/{id}   — eliminar un movimiento y revertir su efecto en stock
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/inventory/stock", h.handleListStock)
	mux.HandleFunc("GET /api/v1/inventory/movements", h.handleListMovements)
	mux.HandleFunc("POST /api/v1/inventory/movements", h.handleMove)
	mux.HandleFunc("DELETE /api/v1/inventory/movements/{id}", h.handleDeleteMovement)
}
