package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/plans"
	"github.com/diegofxm/apidian/internal/subscriptions"
	"github.com/google/uuid"
)

// ── Middleware superadmin ────────────────────────────────────────────────────────────────────

// requireSuperAdmin es un wrapper de handler que devuelve 403 si el usuario no es superadmin.
func (a *API) requireSuperAdmin(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		ok, err := a.auth.IsSuperAdmin(r.Context(), userID)
		if err != nil {
			response.WriteError(w, err)
			return
		}
		if !ok {
			response.WriteJSON(w, http.StatusForbidden, response.Error{Error: "acceso restringido a superadministradores"})
			return
		}
		h(w, r)
	}
}

// ── DTOs ────────────────────────────────────────────────────────────────────────────────────

type planResponse struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	MaxDocumentsPerMonth *int       `json:"max_documents_per_month"`
	MaxIssuers           int        `json:"max_issuers"`
	PriceCOP             int        `json:"price_cop"`
	IsActive             bool       `json:"is_active"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func planToResponse(p *plans.Plan) planResponse {
	return planResponse{
		ID:                   p.ID,
		Name:                 p.Name,
		Description:          p.Description,
		MaxDocumentsPerMonth: p.MaxDocumentsPerMonth,
		MaxIssuers:           p.MaxIssuers,
		PriceCOP:             p.PriceCOP,
		IsActive:             p.IsActive,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

// ── Planes ──────────────────────────────────────────────────────────────────────────────────

func (a *API) handleAdminListPlans(w http.ResponseWriter, r *http.Request) {
	list, err := a.plans.List(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	out := make([]planResponse, len(list))
	for i := range list {
		out[i] = planToResponse(&list[i])
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"plans": out, "count": len(out)})
}

type createPlanRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	MaxDocumentsPerMonth *int   `json:"max_documents_per_month"`
	MaxIssuers           int    `json:"max_issuers"`
	PriceCOP             int    `json:"price_cop"`
}

func (a *API) handleAdminCreatePlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}
	if req.Name == "" {
		response.WriteJSON(w, http.StatusUnprocessableEntity, response.Error{Error: "name es requerido"})
		return
	}
	p, err := a.plans.Create(r.Context(), plans.Plan{
		Name:                 req.Name,
		Description:          req.Description,
		MaxDocumentsPerMonth: req.MaxDocumentsPerMonth,
		MaxIssuers:           req.MaxIssuers,
		PriceCOP:             req.PriceCOP,
		IsActive:             true,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, planToResponse(p))
}

type updatePlanRequest struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	MaxDocumentsPerMonth *int   `json:"max_documents_per_month"`
	MaxIssuers           int    `json:"max_issuers"`
	PriceCOP             int    `json:"price_cop"`
	IsActive             bool   `json:"is_active"`
}

func (a *API) handleAdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	existing, err := a.plans.GetByID(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	var req updatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.MaxDocumentsPerMonth = req.MaxDocumentsPerMonth
	existing.MaxIssuers = req.MaxIssuers
	existing.PriceCOP = req.PriceCOP
	existing.IsActive = req.IsActive

	p, err := a.plans.Update(r.Context(), *existing)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, planToResponse(p))
}

// ── Suscripciones por emisor ─────────────────────────────────────────────────────────────────

func (a *API) handleAdminGetIssuerSubscription(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	sub, err := a.subscriptions.GetActive(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, sub)
}

func (a *API) handleAdminAssignPlan(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		PlanID uuid.UUID `json:"plan_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}
	if body.PlanID == uuid.Nil {
		response.WriteJSON(w, http.StatusUnprocessableEntity, response.Error{Error: "plan_id es requerido"})
		return
	}
	sub, err := a.subscriptions.Assign(r.Context(), issuerID, body.PlanID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, sub)
}

// handleAdminGetIssuer permite al superadmin consultar cualquier emisor por ID.
func (a *API) handleAdminGetIssuer(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	iss, err := a.issuers.GetIssuer(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, issuerToResponse(iss))
}

// handleAdminBillingSummary devuelve el resumen de facturación del mes actual por emisor.
func (a *API) handleAdminBillingSummary(w http.ResponseWriter, r *http.Request) {
	entries, err := a.subscriptions.BillingSummary(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if entries == nil {
		entries = []subscriptions.BillingEntry{}
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"entries": entries, "count": len(entries)})
}
