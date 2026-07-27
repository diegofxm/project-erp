package api

import (
	"net/http"

	"github.com/diegofxm/apidian/internal/api/response"
)

// Handlers de solo lectura para internal/catalogs — todos siguen el mismo patrón (sin
// Service, ver internal/catalogs/model.go) así que no hay DTOs propios: catalogs.Entry/
// Municipality/Currency ya tienen tags json listos para responder tal cual.

func (a *API) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListDepartments(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"departments": out, "count": len(out)})
}

// handleListMunicipalities soporta ?department_code= para no forzar a transferir y filtrar
// 1.122 filas en el cliente cuando solo hace falta un select dependiente departamento→municipio.
func (a *API) handleListMunicipalities(w http.ResponseWriter, r *http.Request) {
	departmentCode := r.URL.Query().Get("department_code")
	out, err := a.catalogs.ListMunicipalities(r.Context(), departmentCode)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"municipalities": out, "count": len(out)})
}

func (a *API) handleListIdentificationTypes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListIdentificationTypes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"identification_types": out, "count": len(out)})
}

func (a *API) handleListTaxTypes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListTaxTypes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"tax_types": out, "count": len(out)})
}

// handleListPaymentMethods — Medios de Pago (cbc:PaymentMeansCode). No confundir con
// handleListPaymentTerms (Formas de Pago, cbc:PaymentMeans/ID) — son catálogos distintos, ver
// migrations/000002_catalogs.up.sql.
func (a *API) handleListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListPaymentMethods(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"payment_methods": out, "count": len(out)})
}

func (a *API) handleListPaymentTerms(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListPaymentTerms(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"payment_terms": out, "count": len(out)})
}

func (a *API) handleListUnitMeasures(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListUnitMeasures(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"unit_measures": out, "count": len(out)})
}

func (a *API) handleListTaxRegimes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListTaxRegimes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"tax_regimes": out, "count": len(out)})
}

func (a *API) handleListLiabilityCodes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListLiabilityCodes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"liability_codes": out, "count": len(out)})
}

func (a *API) handleListDianDocumentTypes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListDianDocumentTypes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"dian_document_types": out, "count": len(out)})
}

func (a *API) handleListCurrencies(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListCurrencies(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"currencies": out, "count": len(out)})
}

// handleListItemStandards — selector de @schemeID/@schemeName/@schemeAgencyID para
// cac:StandardItemIdentification (tabla 13.3.5, ver docs/apidian-architecture.md sección
// 9.45). Exactamente 4 filas fijas, no confundir con un catálogo completo de códigos
// UNSPSC/GTIN/Arancel (esos no se cargan, ver sección 9.45).
func (a *API) handleListItemStandards(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListItemStandards(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"item_standards": out, "count": len(out)})
}

func (a *API) handleListCiiuCodes(w http.ResponseWriter, r *http.Request) {
	out, err := a.catalogs.ListCiiuCodes(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"ciiu_codes": out, "count": len(out)})
}
