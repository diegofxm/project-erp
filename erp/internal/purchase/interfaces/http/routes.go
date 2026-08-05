package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/purchases", h.handleList)
	mux.HandleFunc("POST /api/v1/purchases", h.handleCreate)
	mux.HandleFunc("GET /api/v1/purchases/{id}", h.handleGetByID)
	mux.HandleFunc("POST /api/v1/purchases/{id}/confirm", h.handleConfirm)
	mux.HandleFunc("POST /api/v1/purchases/{id}/receive", h.handleReceive)
	mux.HandleFunc("POST /api/v1/purchases/{id}/cancel", h.handleCancel)
	mux.HandleFunc("DELETE /api/v1/purchases/{id}", h.handleDelete)
	mux.HandleFunc("GET /api/v1/purchases/{id}/pdf", h.handleGetPDF)
	mux.HandleFunc("POST /api/v1/purchases/{id}/send-email", h.handleSendEmail)
	mux.HandleFunc("POST /api/v1/purchases/{id}/withholdings", h.handleAddWithholding)
	mux.HandleFunc("GET /api/v1/purchases/{id}/withholdings", h.handleListWithholdings)

	// Pagos y cuentas por pagar
	mux.HandleFunc("POST /api/v1/purchase-payments", h.handleRecordPayment)
	mux.HandleFunc("GET /api/v1/purchase-payments/purchases/{purchase_id}", h.handleListPayments)
	mux.HandleFunc("GET /api/v1/payables", h.handleGetPayables)

	// Consecutivos (migración de numeración previa)
	mux.HandleFunc("POST /api/v1/purchase/number-counters", h.handleSetNumberCounter)
}
