package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Cuentas (PUC — globales, sin company_id)
	mux.HandleFunc("GET /api/v1/accounting/accounts", h.handleListAccounts)
	mux.HandleFunc("GET /api/v1/accounting/accounts/{code}", h.handleGetAccount)

	// Períodos contables
	mux.HandleFunc("GET /api/v1/accounting/periods", h.handleListPeriods)
	mux.HandleFunc("POST /api/v1/accounting/periods/{id}/close", h.handleClosePeriod)

	// Asientos contables
	mux.HandleFunc("GET /api/v1/accounting/journals", h.handleListJournals)
	mux.HandleFunc("POST /api/v1/accounting/journals", h.handlePostJournal)
	mux.HandleFunc("GET /api/v1/accounting/journals/{id}", h.handleGetJournal)
	mux.HandleFunc("POST /api/v1/accounting/journals/{id}/void", h.handleVoidJournal)

	// Reportes financieros
	mux.HandleFunc("GET /api/v1/accounting/reports/pl", h.handlePLReport)
	mux.HandleFunc("GET /api/v1/accounting/reports/bs", h.handleBSReport)
	mux.HandleFunc("GET /api/v1/accounting/reports/trial-balance", h.handleTrialBalance)
	mux.HandleFunc("GET /api/v1/accounting/reports/ledger/{code}", h.handleAccountLedger)
}
