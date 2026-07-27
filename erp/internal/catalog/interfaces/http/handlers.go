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
	respond(w, data, err)
}

func (h *Handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListDepartments(r.Context())
	respond(w, data, err)
}

func (h *Handler) listMunicipalities(w http.ResponseWriter, r *http.Request) {
	dept := r.URL.Query().Get("department")
	data, err := h.repo.ListMunicipalities(r.Context(), dept)
	respond(w, data, err)
}

func (h *Handler) listIdentificationTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListIdentificationTypes(r.Context())
	respond(w, data, err)
}

func (h *Handler) listTaxTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListTaxTypes(r.Context())
	respond(w, data, err)
}

func (h *Handler) listPaymentMethods(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPaymentMethods(r.Context())
	respond(w, data, err)
}

func (h *Handler) listPaymentTerms(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListPaymentTerms(r.Context())
	respond(w, data, err)
}

func (h *Handler) listUnitMeasures(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListUnitMeasures(r.Context())
	respond(w, data, err)
}

func (h *Handler) listTaxRegimes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListTaxRegimes(r.Context())
	respond(w, data, err)
}

func (h *Handler) listLiabilityCodes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListLiabilityCodes(r.Context())
	respond(w, data, err)
}

func (h *Handler) listDocumentTypes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListDianDocumentTypes(r.Context())
	respond(w, data, err)
}

func (h *Handler) listItemStandards(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListItemStandards(r.Context())
	respond(w, data, err)
}

func (h *Handler) listCiiuCodes(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.ListCiiuCodes(r.Context())
	respond(w, data, err)
}

func respond(w http.ResponseWriter, data any, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
