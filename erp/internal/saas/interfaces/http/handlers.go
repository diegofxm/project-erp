package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/application"
	"github.com/diegofxm/erp/internal/saas/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría — mismo patrón duck-typed
// que el resto de módulos (ver accounting/interfaces/http).
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

type Handler struct {
	plans     *application.PlanUseCase
	settings  *application.SettingsUseCase
	subs      *application.SubscriptionUseCase
	billing   *application.BillingUseCase
	payments  *application.PaymentUseCase
	prospects *application.ProspectUseCase
	myPlan    *application.MyPlanUseCase
	users     domain.UserPort
	company   domain.CompanyPort
	audit     AuditLogger
}

func NewHandler(
	plans *application.PlanUseCase,
	settings *application.SettingsUseCase,
	subs *application.SubscriptionUseCase,
	billing *application.BillingUseCase,
	payments *application.PaymentUseCase,
	prospects *application.ProspectUseCase,
	myPlan *application.MyPlanUseCase,
	users domain.UserPort,
	company domain.CompanyPort,
	audit AuditLogger,
) *Handler {
	return &Handler{
		plans: plans, settings: settings, subs: subs, billing: billing, payments: payments,
		prospects: prospects, myPlan: myPlan, users: users, company: company, audit: audit,
	}
}

func (h *Handler) logAudit(ctx context.Context, action, resourceType string, id uuid.UUID, meta map[string]any) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	// Acciones de plataforma no tienen company_id propio — se usa uuid.Nil, igual que el resto de
	// módulos usa &id para el recurso afectado (que sí suele ser una empresa cliente).
	h.audit.Log(ctx, uuid.Nil, userID, action, resourceType, &id, meta)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func requireSuperAdmin(w http.ResponseWriter, r *http.Request) bool {
	if tenant.GetUserID(r.Context()) == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "autenticación requerida")
		return false
	}
	if !tenant.IsSuperAdmin(r.Context()) {
		respondError(w, http.StatusForbidden, "requiere superadmin")
		return false
	}
	return true
}

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// writeErr mapea errores de dominio conocidos a su status HTTP — el resto cae a 500.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrPlanNotFound), errors.Is(err, domain.ErrModuleNotFound),
		errors.Is(err, domain.ErrSubscriptionNotFound), errors.Is(err, domain.ErrProspectNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrPlanInactive), errors.Is(err, domain.ErrProspectNotPending),
		errors.Is(err, domain.ErrEmailTaken):
		respondError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, err.Error())
	}
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
