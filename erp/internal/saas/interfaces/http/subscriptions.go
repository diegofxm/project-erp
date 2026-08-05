package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/application"
	"github.com/diegofxm/erp/internal/saas/domain"
)

func companyIDFromPath(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id de empresa inválido")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) handleGetCompanyInfo(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	c, err := h.company.GetCompany(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusNotFound, "empresa no encontrada")
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"id": c.ID, "business_name": c.BusinessName, "trade_name": c.TradeName, "nit": c.NIT,
	})
}

func (h *Handler) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	s, err := h.subs.Get(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond(w, http.StatusOK, toSubscriptionDTO(s))
}

func (h *Handler) handleAssignSubscription(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		PlanID            string `json:"plan_id"`
		HasOwnCertificate bool   `json:"has_own_certificate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	planID, err := uuid.Parse(body.PlanID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "plan_id inválido")
		return
	}
	s, err := h.subs.Assign(r.Context(), companyID, planID, body.HasOwnCertificate)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.subscription_assigned", "subscription", s.ID, map[string]any{
		"company_id": companyID, "plan_id": planID,
	})
	respond(w, http.StatusCreated, toSubscriptionDTO(s))
}

func (h *Handler) handleRenewSubscription(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	s, err := h.subs.Renew(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.subscription_renewed", "subscription", s.ID, map[string]any{
		"new_period_end": s.CurrentPeriodEnd,
	})
	respond(w, http.StatusOK, toSubscriptionDTO(s))
}

func (h *Handler) handleBillingSummary(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	list, err := h.billing.Summary(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toBillingEntryDTOs(list)
	respond(w, http.StatusOK, map[string]any{"entries": dtos, "count": len(dtos)})
}

func (h *Handler) handleBillingRenewals(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	withinDays := 90
	if v := r.URL.Query().Get("within_days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			withinDays = n
		}
	}
	list, err := h.billing.Renewals(r.Context(), withinDays)
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toRenewalEntryDTOs(list)
	respond(w, http.StatusOK, map[string]any{"entries": dtos, "count": len(dtos)})
}

func (h *Handler) handleListPayments(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	list, err := h.payments.ListByCompany(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toPaymentDTOs(list)
	respond(w, http.StatusOK, map[string]any{"payments": dtos, "count": len(dtos)})
}

func (h *Handler) handleRecordPayment(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	companyID, ok := companyIDFromPath(w, r)
	if !ok {
		return
	}
	var body struct {
		SubscriptionID string `json:"subscription_id"`
		Type           string `json:"type"`
		AmountCents    int64  `json:"amount_cents"`
		Note           string `json:"note"`
		PaidAt         string `json:"paid_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	req := application.RecordPaymentRequest{
		CompanyID: companyID, Type: domain.PaymentType(body.Type), AmountCents: body.AmountCents,
		Note: body.Note,
	}
	if body.PaidAt != "" {
		req.PaidAt = parseDate(body.PaidAt)
	}
	if body.SubscriptionID != "" {
		if subID, err := uuid.Parse(body.SubscriptionID); err == nil {
			req.SubscriptionID = &subID
		}
	}
	p, err := h.payments.Record(r.Context(), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.payment_recorded", "payment", p.ID, map[string]any{
		"company_id": companyID, "amount_cents": p.AmountCents, "type": p.Type,
	})
	respond(w, http.StatusCreated, toPaymentDTO(p))
}
