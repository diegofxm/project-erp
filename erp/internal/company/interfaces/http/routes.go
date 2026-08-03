package http

import "net/http"

// RegisterRoutes monta las rutas del módulo company.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/companies", h.handleListForUser)
	mux.HandleFunc("POST /api/v1/companies", h.handleCreate)
	mux.HandleFunc("GET /api/v1/companies/active", h.handleGetActive)
	mux.HandleFunc("PUT /api/v1/companies/active", h.handleUpdateProfile)
	mux.HandleFunc("PUT /api/v1/companies/active/credentials", h.handleUpdateCredentials)
	mux.HandleFunc("DELETE /api/v1/companies/active/credentials/software", h.handleClearSoftware)
	mux.HandleFunc("DELETE /api/v1/companies/active/credentials/ne-software", h.handleClearNeSoftware)
	mux.HandleFunc("DELETE /api/v1/companies/active/credentials/certificate", h.handleClearCertificate)
	mux.HandleFunc("GET /api/v1/companies/active/settings", h.handleGetSettings)
	mux.HandleFunc("PATCH /api/v1/companies/active/settings", h.handleUpdateSettings)
	mux.HandleFunc("GET /api/v1/companies/active/logo", h.handleGetLogo)
	mux.HandleFunc("PUT /api/v1/companies/active/logo", h.handleUpdateLogo)
	mux.HandleFunc("DELETE /api/v1/companies/active/logo", h.handleDeleteLogo)

	// Bodegas
	mux.HandleFunc("GET /api/v1/companies/active/warehouses", h.handleListWarehouses)
	mux.HandleFunc("POST /api/v1/companies/active/warehouses", h.handleCreateWarehouse)
	mux.HandleFunc("GET /api/v1/companies/active/warehouses/{id}", h.handleGetWarehouse)
	mux.HandleFunc("PUT /api/v1/companies/active/warehouses/{id}", h.handleUpdateWarehouse)
	mux.HandleFunc("DELETE /api/v1/companies/active/warehouses/{id}", h.handleDeactivateWarehouse)
	mux.HandleFunc("PUT /api/v1/companies/active/warehouses/{id}/default", h.handleSetDefaultWarehouse)
}
