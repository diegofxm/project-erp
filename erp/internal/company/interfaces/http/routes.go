package http

import "net/http"

// RegisterRoutes monta las rutas del módulo company.
//
//   POST   /api/v1/companies            — crear empresa (auth, sin tenant)
//   GET    /api/v1/companies/active     — obtener empresa activa (tenant)
//   PUT    /api/v1/companies/active     — actualizar perfil (tenant)
//   PUT    /api/v1/companies/active/credentials — actualizar credenciales DIAN (tenant)
//   PUT    /api/v1/companies/active/logo        — subir logo (tenant, body = bytes raw)
//   DELETE /api/v1/companies/active/logo        — eliminar logo (tenant)
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/companies", h.handleCreate)
	mux.HandleFunc("GET /api/v1/companies/active", h.handleGetActive)
	mux.HandleFunc("PUT /api/v1/companies/active", h.handleUpdateProfile)
	mux.HandleFunc("PUT /api/v1/companies/active/credentials", h.handleUpdateCredentials)
	mux.HandleFunc("PUT /api/v1/companies/active/logo", h.handleUpdateLogo)
	mux.HandleFunc("DELETE /api/v1/companies/active/logo", h.handleDeleteLogo)

	// Bodegas
	mux.HandleFunc("GET /api/v1/companies/active/warehouses", h.handleListWarehouses)
	mux.HandleFunc("POST /api/v1/companies/active/warehouses", h.handleCreateWarehouse)
	mux.HandleFunc("GET /api/v1/companies/active/warehouses/{id}", h.handleGetWarehouse)
	mux.HandleFunc("PUT /api/v1/companies/active/warehouses/{id}", h.handleUpdateWarehouse)
	mux.HandleFunc("DELETE /api/v1/companies/active/warehouses/{id}", h.handleDeactivateWarehouse)
}
