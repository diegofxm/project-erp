package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sales", h.handleList)
	mux.HandleFunc("POST /api/v1/sales", h.handleCreate)
	mux.HandleFunc("GET /api/v1/sales/{id}", h.handleGetByID)
	mux.HandleFunc("POST /api/v1/sales/{id}/confirm", h.handleConfirm)
	mux.HandleFunc("POST /api/v1/sales/{id}/cancel", h.handleCancel)

	// Cotizaciones
	mux.HandleFunc("GET /api/v1/quotes", h.handleListQuotes)
	mux.HandleFunc("POST /api/v1/quotes", h.handleCreateQuote)
	mux.HandleFunc("GET /api/v1/quotes/{id}", h.handleGetQuote)
	mux.HandleFunc("DELETE /api/v1/quotes/{id}", h.handleDeleteQuote)
	mux.HandleFunc("POST /api/v1/quotes/{id}/send", h.handleSendQuote)
	mux.HandleFunc("GET /api/v1/quotes/{id}/pdf", h.handleGetQuotePDF)
	mux.HandleFunc("POST /api/v1/quotes/{id}/send-email", h.handleSendQuoteEmail)
	mux.HandleFunc("POST /api/v1/quotes/{id}/accept", h.handleAcceptQuote)
	mux.HandleFunc("POST /api/v1/quotes/{id}/reject", h.handleRejectQuote)
	mux.HandleFunc("POST /api/v1/quotes/{id}/convert-to-sale", h.handleConvertQuoteToSale)

	// Pagos y cartera
	mux.HandleFunc("POST /api/v1/payments", h.handleRecordPayment)
	mux.HandleFunc("GET /api/v1/payments/sales/{sale_id}", h.handleListPayments)
	mux.HandleFunc("GET /api/v1/receivables", h.handleGetReceivables)

	// Consecutivos (migración de numeración previa)
	mux.HandleFunc("POST /api/v1/sales/number-counters", h.handleSetNumberCounter)
}
