package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

func (h *Handler) handleListModules(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	list, err := h.plans.ListModules(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]any{"modules": toModuleDTOs(list)})
}

func (h *Handler) handleListPlans(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	list, err := h.plans.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toPlanDTOs(list)
	respond(w, http.StatusOK, map[string]any{"plans": dtos, "count": len(dtos)})
}

type planBody struct {
	Code                       string   `json:"code"`
	Name                       string   `json:"name"`
	Description                string   `json:"description"`
	BillingCycle               string   `json:"billing_cycle"`
	PriceCents                 int64    `json:"price_cents"`
	IncludedDocuments          *int     `json:"included_documents"`
	PricePerExtraDocumentCents int64    `json:"price_per_extra_document_cents"`
	RequiresCertificate        bool     `json:"requires_certificate"`
	CertificatePriceCents      int64    `json:"certificate_price_cents"`
	AnnualIncrementPct         float64  `json:"annual_increment_pct"`
	IsInternal                 bool     `json:"is_internal"`
	IsActive                   bool     `json:"is_active"`
	Modules                    []string `json:"modules"`
}

func (h *Handler) handleCreatePlan(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	var body planBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.plans.Create(r.Context(), domain.Plan{
		Code: body.Code, Name: body.Name, Description: body.Description,
		BillingCycle: domain.BillingCycle(body.BillingCycle), PriceCents: body.PriceCents,
		IncludedDocuments: body.IncludedDocuments, PricePerExtraDocumentCents: body.PricePerExtraDocumentCents,
		RequiresCertificate: body.RequiresCertificate, CertificatePriceCents: body.CertificatePriceCents,
		AnnualIncrementPct: body.AnnualIncrementPct, IsInternal: body.IsInternal, IsActive: true,
		ModuleCodes: body.Modules,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.plan_created", "plan", p.ID, map[string]any{"code": p.Code})
	respond(w, http.StatusCreated, toPlanDTO(p))
}

func (h *Handler) handleUpdatePlan(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body planBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	p, err := h.plans.Update(r.Context(), domain.Plan{
		ID: id, Name: body.Name, Description: body.Description,
		BillingCycle: domain.BillingCycle(body.BillingCycle), PriceCents: body.PriceCents,
		IncludedDocuments: body.IncludedDocuments, PricePerExtraDocumentCents: body.PricePerExtraDocumentCents,
		RequiresCertificate: body.RequiresCertificate, CertificatePriceCents: body.CertificatePriceCents,
		AnnualIncrementPct: body.AnnualIncrementPct, IsActive: body.IsActive,
		ModuleCodes: body.Modules,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.plan_updated", "plan", p.ID, map[string]any{"is_active": p.IsActive})
	respond(w, http.StatusOK, toPlanDTO(p))
}

func (h *Handler) handleApplyIncrement(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.plans.ApplyIncrement(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.plan_increment_applied", "plan", p.ID, map[string]any{"new_price_cents": p.PriceCents})
	respond(w, http.StatusOK, toPlanDTO(p))
}

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	s, err := h.settings.Get(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	respond(w, http.StatusOK, toSettingsDTO(s))
}

func (h *Handler) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	var body struct {
		IVARateBP int `json:"iva_rate_bp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	s, err := h.settings.Update(r.Context(), domain.Settings{IVARateBP: body.IVARateBP})
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.settings_updated", "settings", uuid.Nil, map[string]any{"iva_rate_bp": s.IVARateBP})
	respond(w, http.StatusOK, toSettingsDTO(s))
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	list, err := h.users.ListAll(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toPlatformUserDTOs(list)
	respond(w, http.StatusOK, map[string]any{"users": dtos, "count": len(dtos)})
}

// handleSetUserSuperAdmin promueve/degrada el flag de superadmin de un usuario — nunca se otorga
// por ningún flujo de alta de cliente (registro, invitación, aprobación de prospecto), solo un
// superadmin existente puede otorgarlo a otro usuario.
func (h *Handler) handleSetUserSuperAdmin(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		IsSuperAdmin bool `json:"is_superadmin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if err := h.users.SetSuperAdmin(r.Context(), id, body.IsSuperAdmin); err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.user_superadmin_set", "user", id, map[string]any{"is_superadmin": body.IsSuperAdmin})
	respond(w, http.StatusOK, map[string]any{"id": id, "is_superadmin": body.IsSuperAdmin})
}
