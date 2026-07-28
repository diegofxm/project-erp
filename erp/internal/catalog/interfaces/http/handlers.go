package http

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/erp/internal/catalog/domain"
)

// Handler expone los catálogos DIAN/DANE como endpoints de solo lectura.
type Handler struct {
	repo domain.Repository
}

func NewHandler(repo domain.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) listCurrencies(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListCurrencies(r.Context())
	respondKeyed(w, "currencies", data, err)
}

func (h *Handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListDepartments(r.Context())
	respondKeyed(w, "departments", data, err)
}

func (h *Handler) listMunicipalities(w http.ResponseWriter, r *http.Request) {
	dept := r.URL.Query().Get("department_code")
	if dept == "" {
		dept = r.URL.Query().Get("department")
	}
	data, err := h.repo.ListMunicipalities(r.Context(), dept)
	respondKeyed(w, "municipalities", data, err)
}

func (h *Handler) listIdentificationTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListIdentificationTypes(r.Context())
	respondKeyed(w, "identification_types", data, err)
}

func (h *Handler) listTaxTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListTaxTypes(r.Context())
	respondKeyed(w, "tax_types", data, err)
}

func (h *Handler) listPaymentMethods(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPaymentMethods(r.Context())
	respondKeyed(w, "payment_methods", data, err)
}

func (h *Handler) listPaymentTerms(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPaymentTerms(r.Context())
	respondKeyed(w, "payment_terms", data, err)
}

func (h *Handler) listUnitMeasures(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListUnitMeasures(r.Context())
	respondKeyed(w, "unit_measures", data, err)
}

func (h *Handler) listTaxRegimes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListTaxRegimes(r.Context())
	respondKeyed(w, "tax_regimes", data, err)
}

func (h *Handler) listLiabilityCodes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListLiabilityCodes(r.Context())
	respondKeyed(w, "liability_codes", data, err)
}

func (h *Handler) listDocumentTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListDianDocumentTypes(r.Context())
	respondKeyed(w, "document_types", data, err)
}

func (h *Handler) listItemStandards(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListItemStandards(r.Context())
	respondKeyed(w, "item_standards", data, err)
}

func (h *Handler) listCiiuCodes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListCiiuCodes(r.Context())
	respondKeyed(w, "ciiu_codes", data, err)
}

// respondKeyed envuelve la lista en un objeto {key: [...]} — patrón estándar del API.
func respondKeyed(w http.ResponseWriter, key string, data any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{key: data})
}
