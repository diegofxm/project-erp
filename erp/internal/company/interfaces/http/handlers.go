package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/application"
	"github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	create    *application.CreateUseCase
	get       *application.GetUseCase
	profile   *application.UpdateProfileUseCase
	creds     *application.UpdateCredentialsUseCase
	clearCred *application.ClearCredentialUseCase
	logo      *application.UpdateLogoUseCase
	warehouse *application.WarehouseUseCase
}

func NewHandler(
	create *application.CreateUseCase,
	get *application.GetUseCase,
	profile *application.UpdateProfileUseCase,
	creds *application.UpdateCredentialsUseCase,
	clearCred *application.ClearCredentialUseCase,
	logo *application.UpdateLogoUseCase,
	warehouse *application.WarehouseUseCase,
) *Handler {
	return &Handler{
		create: create, get: get, profile: profile,
		creds: creds, clearCred: clearCred,
		logo: logo, warehouse: warehouse,
	}
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}

	var req application.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	c, err := h.create.Execute(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, domain.ErrNITTaken) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, safeCompany(c))
}

func (h *Handler) handleListForUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	companies, err := h.get.ListByUserID(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, len(companies))
	for i := range companies {
		out[i] = safeCompany(&companies[i])
	}
	respond(w, http.StatusOK, out)
}

func (h *Handler) handleGetActive(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var req application.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	c, err := h.profile.Execute(r.Context(), companyID, req)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var body struct {
		SoftwareID          string `json:"software_id"`
		SoftwarePIN         string `json:"software_pin"`
		CertificateBase64   string `json:"certificate_base64"`
		CertificatePassword string `json:"certificate_password"`
		NeSoftwareID        string `json:"ne_software_id"`
		NeSoftwarePIN       string `json:"ne_software_pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	var certBytes []byte
	if body.CertificateBase64 != "" {
		var err error
		certBytes, err = base64.StdEncoding.DecodeString(body.CertificateBase64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "certificate_base64 inválido")
			return
		}
	}

	req := application.UpdateCredentialsRequest{
		SoftwareID:          body.SoftwareID,
		SoftwarePIN:         body.SoftwarePIN,
		Certificate:         certBytes,
		CertificatePassword: body.CertificatePassword,
		NeSoftwareID:        body.NeSoftwareID,
		NeSoftwarePIN:       body.NeSoftwarePIN,
	}

	if err := h.creds.Execute(r.Context(), companyID, req); err != nil {
		if errors.Is(err, application.ErrBadCertificate) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleClearSoftware(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	if err := h.clearCred.Execute(r.Context(), companyID, application.CredentialSoftware); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleClearNeSoftware(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	if err := h.clearCred.Execute(r.Context(), companyID, application.CredentialNeSoftware); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleClearCertificate(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	if err := h.clearCred.Execute(r.Context(), companyID, application.CredentialCertificate); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleGetLogo(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(c.Logo) == 0 {
		respondError(w, http.StatusNotFound, "sin logo")
		return
	}
	ct := c.LogoContentType
	if ct == "" {
		ct = "image/png"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(c.Logo)
}

func (h *Handler) handleUpdateLogo(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var body struct {
		LogoBase64  string `json:"logo_base64"`
		ContentType string `json:"logo_content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}

	data, err := base64.StdEncoding.DecodeString(body.LogoBase64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "logo_base64 inválido")
		return
	}

	if err := h.logo.Update(r.Context(), companyID, data, body.ContentType); err != nil {
		if errors.Is(err, application.ErrInvalidLogoType) {
			respondError(w, http.StatusUnsupportedMediaType, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, domain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"brand_color": c.BrandColor,
	})
}

func (h *Handler) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var body struct {
		BrandColor string `json:"brand_color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := h.get.UpdateBrandColor(r.Context(), companyID, body.BrandColor); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"brand_color": body.BrandColor,
	})
}

func (h *Handler) handleDeleteLogo(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	if err := h.logo.Delete(r.Context(), companyID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	c, err := h.get.ByID(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, safeCompany(c))
}

// ── Bodegas ───────────────────────────────────────────────────────────────────────────────

func (h *Handler) handleCreateWarehouse(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.CreateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	wh, err := h.warehouse.Create(r.Context(), cid, req)
	if err != nil {
		if errors.Is(err, domain.ErrWarehouseCodeTaken) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusCreated, wh)
}

func (h *Handler) handleListWarehouses(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.warehouse.List(r.Context(), cid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []domain.Warehouse{}
	}
	respond(w, http.StatusOK, list)
}

func (h *Handler) handleGetWarehouse(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	wh, err := h.warehouse.GetByID(r.Context(), cid, id)
	if err != nil {
		if errors.Is(err, domain.ErrWarehouseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, wh)
}

func (h *Handler) handleUpdateWarehouse(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var req application.UpdateWarehouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	wh, err := h.warehouse.Update(r.Context(), cid, id, req)
	if err != nil {
		if errors.Is(err, domain.ErrWarehouseNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, domain.ErrWarehouseCodeTaken) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, wh)
}

func (h *Handler) handleDeactivateWarehouse(w http.ResponseWriter, r *http.Request) {
	cid, ok := requireTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	if err := h.warehouse.Deactivate(r.Context(), cid, id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "deactivated"})
}

// --- helpers ---

func requireAuth(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	uid := tenant.GetUserID(r.Context())
	if uid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "autenticación requerida")
		return uuid.Nil, false
	}
	return uid, true
}

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func safeCompany(c *domain.Company) map[string]any {
	return map[string]any{
		"id":                            c.ID,
		"nit":                           c.NIT,
		"check_digit":                   c.CheckDigit,
		"business_name":                 c.BusinessName,
		"trade_name":                    c.TradeName,
		"identification_type_code":      c.IdentificationTypeCode,
		"department_code":               c.DepartmentCode,
		"municipality_code":             c.MunicipalityCode,
		"address_line":                  c.AddressLine,
		"email":                         c.Email,
		"phone":                         c.Phone,
		"environment":                   c.Environment,
		"entity_type_code":              c.EntityTypeCode,
		"tax_scheme_code":               c.TaxSchemeCode,
		"tax_scheme_name":               c.TaxSchemeName,
		"liability_codes":               c.LiabilityCodes,
		"tax_regime_code":               c.TaxRegimeCode,
		"industry_classification_codes": c.IndustryClassificationCodes,
		"merchant_registration_number":  c.MerchantRegistrationNumber,
		"software_id":                   c.SoftwareID,
		"has_software_credentials":      c.SoftwareID != "",
		"ne_software_id":                c.NeSoftwareID,
		"has_ne_software_credentials":   c.NeSoftwareID != "",
		"has_certificate":               len(c.Certificate) > 0,
		"certificate_subject":           c.CertificateSubject,
		"certificate_issuer_cn":         c.CertificateIssuerCN,
		"certificate_expires_at":        c.CertificateExpiresAt,
		"has_logo":                      len(c.Logo) > 0,
		"logo_content_type":             c.LogoContentType,
		"is_active":                     c.IsActive,
		"created_at":                    c.CreatedAt,
		"updated_at":                    c.UpdatedAt,
	}
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
