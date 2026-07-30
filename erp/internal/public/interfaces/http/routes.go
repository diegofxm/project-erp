package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/public/companies/{id}", h.handleGetIssuer)
	mux.HandleFunc("GET /api/v1/public/companies/{id}/logo", h.handleGetIssuerLogo)
	mux.HandleFunc("POST /api/v1/public/companies/{id}/customers", h.handleRegisterCustomer)
}
