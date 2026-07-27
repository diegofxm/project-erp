package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Públicas
	mux.HandleFunc("POST /api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/accept-invite", h.handleAcceptInvite)

	// Protegidas (el middleware de tenant verifica el JWT; el handler llama requireAuth)
	mux.HandleFunc("GET /api/v1/auth/me", h.handleMe)
	mux.HandleFunc("PUT /api/v1/auth/profile", h.handleUpdateProfile)
	mux.HandleFunc("POST /api/v1/auth/select-company", h.handleSelectCompany)
	mux.HandleFunc("POST /api/v1/auth/invite", h.handleInvite)
	mux.HandleFunc("GET /api/v1/auth/companies", h.handleListCompanies)
}
