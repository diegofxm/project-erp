package http

import "net/http"

// RegisterAdminRoutes registra las rutas de superadmin — transversales a todas las empresas, no
// dependen de company_id activo. Cada handler valida requireSuperAdmin internamente.
func (h *Handler) RegisterAdminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/modules", h.handleListModules)

	mux.HandleFunc("GET /api/v1/admin/plans", h.handleListPlans)
	mux.HandleFunc("POST /api/v1/admin/plans", h.handleCreatePlan)
	mux.HandleFunc("PATCH /api/v1/admin/plans/{id}", h.handleUpdatePlan)
	mux.HandleFunc("POST /api/v1/admin/plans/{id}/apply-increment", h.handleApplyIncrement)

	mux.HandleFunc("GET /api/v1/admin/settings", h.handleGetSettings)
	mux.HandleFunc("PATCH /api/v1/admin/settings", h.handleUpdateSettings)

	mux.HandleFunc("GET /api/v1/admin/companies/{id}", h.handleGetCompanyInfo)
	mux.HandleFunc("GET /api/v1/admin/companies/{id}/subscription", h.handleGetSubscription)
	mux.HandleFunc("POST /api/v1/admin/companies/{id}/subscription", h.handleAssignSubscription)
	mux.HandleFunc("POST /api/v1/admin/companies/{id}/subscription/renew", h.handleRenewSubscription)

	mux.HandleFunc("GET /api/v1/admin/billing/summary", h.handleBillingSummary)
	mux.HandleFunc("GET /api/v1/admin/billing/renewals", h.handleBillingRenewals)

	mux.HandleFunc("GET /api/v1/admin/companies/{id}/payments", h.handleListPayments)
	mux.HandleFunc("POST /api/v1/admin/companies/{id}/payments", h.handleRecordPayment)

	mux.HandleFunc("GET /api/v1/admin/users", h.handleListUsers)
	mux.HandleFunc("PATCH /api/v1/admin/users/{id}/superadmin", h.handleSetUserSuperAdmin)

	mux.HandleFunc("GET /api/v1/admin/prospects", h.handleListProspects)
	mux.HandleFunc("POST /api/v1/admin/prospects/{id}/approve", h.handleApproveProspect)
	mux.HandleFunc("POST /api/v1/admin/prospects/{id}/reject", h.handleRejectProspect)
	mux.HandleFunc("GET /api/v1/admin/prospects/{id}/cedula", h.handleDownloadCedula)
	mux.HandleFunc("GET /api/v1/admin/prospects/{id}/rut", h.handleDownloadRUT)
}

// RegisterTenantRoutes registra las rutas de la empresa activa (usuario normal autenticado).
func (h *Handler) RegisterTenantRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/saas/my-plan", h.handleGetMyPlan)
}

// RegisterPublicRoutes registra las rutas sin autenticación — solicitud de acceso a la plataforma.
func (h *Handler) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/public/prospects", h.handleSubmitProspect)
}
