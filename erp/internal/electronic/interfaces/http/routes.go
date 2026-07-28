package http

import "net/http"

// RegisterRoutes registra las rutas del módulo electronic en el mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Documentos
	mux.HandleFunc("GET /api/v1/electronic/documents", h.handleListDocuments)
	mux.HandleFunc("GET /api/v1/electronic/documents/{id}", h.handleGetDocument)
	mux.HandleFunc("GET /api/v1/electronic/documents/{id}/pdf", h.handleGetDocumentPDF)
	mux.HandleFunc("POST /api/v1/electronic/documents/{id}/confirm", h.handleConfirmDocument)
	mux.HandleFunc("DELETE /api/v1/electronic/documents/{id}", h.handleDeleteDraft)

	// Borradores por tipo
	mux.HandleFunc("POST /api/v1/electronic/invoices/from-sale/{sale_id}", h.handleCreateInvoiceFromSale)
	mux.HandleFunc("POST /api/v1/electronic/invoices", h.handleCreateInvoiceDraft)
	mux.HandleFunc("POST /api/v1/electronic/credit-notes", h.handleCreateCreditNoteDraft)
	mux.HandleFunc("POST /api/v1/electronic/debit-notes", h.handleCreateDebitNoteDraft)
	mux.HandleFunc("POST /api/v1/electronic/support-documents", h.handleCreateSupportDocDraft)
	mux.HandleFunc("POST /api/v1/electronic/adjustment-notes", h.handleCreateAdjustmentNoteDraft)

	// Rangos de numeración
	mux.HandleFunc("GET /api/v1/electronic/numbering-ranges", h.handleListNumberingRanges)
	mux.HandleFunc("POST /api/v1/electronic/numbering-ranges", h.handleCreateNumberingRange)
	mux.HandleFunc("DELETE /api/v1/electronic/numbering-ranges/{id}", h.handleDeactivateRange)
	mux.HandleFunc("PUT /api/v1/electronic/numbering-ranges/{id}/activate", h.handleActivateRange)
}
