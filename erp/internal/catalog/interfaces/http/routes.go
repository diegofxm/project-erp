package http

import "net/http"

// RegisterRoutes monta los 13 endpoints de catálogo en el mux del servidor.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/catalogs/currencies", h.listCurrencies)
	mux.HandleFunc("GET /api/v1/catalogs/departments", h.listDepartments)
	mux.HandleFunc("GET /api/v1/catalogs/municipalities", h.listMunicipalities)
	mux.HandleFunc("GET /api/v1/catalogs/identification-types", h.listIdentificationTypes)
	mux.HandleFunc("GET /api/v1/catalogs/tax-types", h.listTaxTypes)
	mux.HandleFunc("GET /api/v1/catalogs/payment-methods", h.listPaymentMethods)
	mux.HandleFunc("GET /api/v1/catalogs/payment-terms", h.listPaymentTerms)
	mux.HandleFunc("GET /api/v1/catalogs/unit-measures", h.listUnitMeasures)
	mux.HandleFunc("GET /api/v1/catalogs/tax-regimes", h.listTaxRegimes)
	mux.HandleFunc("GET /api/v1/catalogs/liability-codes", h.listLiabilityCodes)
	mux.HandleFunc("GET /api/v1/catalogs/document-types", h.listDocumentTypes)
	mux.HandleFunc("GET /api/v1/catalogs/item-standards", h.listItemStandards)
	mux.HandleFunc("GET /api/v1/catalogs/ciiu-codes", h.listCiiuCodes)
}
