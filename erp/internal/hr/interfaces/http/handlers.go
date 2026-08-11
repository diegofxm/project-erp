package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/hr/application"
	"github.com/diegofxm/erp/internal/hr/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

type Handler struct {
	absences application.AbsenceUseCase
	audit    AuditLogger
}

func NewHandler(absences *application.AbsenceUseCase, audit AuditLogger) *Handler {
	return &Handler{absences: *absences, audit: audit}
}

func (h *Handler) logAbsence(ctx context.Context, companyID uuid.UUID, action string, id uuid.UUID, meta map[string]any) {
	if h.audit == nil {
		return
	}
	uid := tenant.GetUserID(ctx)
	var userID *uuid.UUID
	if uid != uuid.Nil {
		userID = &uid
	}
	h.audit.Log(ctx, companyID, userID, "absence."+action, "absence", &id, meta)
}

type requestAbsenceBody struct {
	EmployeeID uuid.UUID         `json:"employee_id"`
	Type       domain.AbsenceType `json:"type"`
	StartDate  string            `json:"start_date"` // YYYY-MM-DD
	EndDate    string            `json:"end_date"`
	Reason     string            `json:"reason"`
}

func (h *Handler) handleRequestAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body requestAbsenceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in := domain.CreateAbsenceInput{
		CompanyID:  companyID,
		EmployeeID: body.EmployeeID,
		Type:       body.Type,
		Reason:     body.Reason,
	}
	if t, err := time.Parse("2006-01-02", body.StartDate); err == nil {
		in.StartDate = t
	}
	if t, err := time.Parse("2006-01-02", body.EndDate); err == nil {
		in.EndDate = t
	}
	a, err := h.absences.Request(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAbsence(r.Context(), companyID, "requested", a.ID, map[string]any{"type": string(a.Type)})
	writeJSON(w, http.StatusCreated, a)
}

// handleUpdateAbsence corrige tipo/fechas/motivo mientras la solicitud sigue pendiente.
func (h *Handler) handleUpdateAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var body requestAbsenceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in := domain.CreateAbsenceInput{CompanyID: companyID, EmployeeID: body.EmployeeID, Type: body.Type, Reason: body.Reason}
	if t, err := time.Parse("2006-01-02", body.StartDate); err == nil {
		in.StartDate = t
	}
	if t, err := time.Parse("2006-01-02", body.EndDate); err == nil {
		in.EndDate = t
	}
	a, err := h.absences.Update(r.Context(), companyID, id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAbsence(r.Context(), companyID, "updated", a.ID, map[string]any{"type": string(a.Type)})
	writeJSON(w, http.StatusOK, a)
}

// handleWithdrawAbsence retira una solicitud pendiente antes de que alguien la revise.
func (h *Handler) handleWithdrawAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.absences.Withdraw(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	h.logAbsence(r.Context(), companyID, "withdrawn", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListAbsences(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	empStr := r.URL.Query().Get("employee_id")
	if empStr != "" {
		empID, err := uuid.Parse(empStr)
		if err != nil {
			http.Error(w, "employee_id inválido", http.StatusBadRequest)
			return
		}
		list, err := h.absences.ListByEmployee(r.Context(), companyID, empID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
		return
	}
	list, err := h.absences.ListByCompany(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) handleGetAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	a, err := h.absences.Get(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

type reviewBody struct {
	Notes string `json:"notes"`
}

func (h *Handler) handleApproveAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireManage(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var body reviewBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.absences.Approve(r.Context(), companyID, id, body.Notes); err != nil {
		writeErr(w, err)
		return
	}
	h.logAbsence(r.Context(), companyID, "approved", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRejectAbsence(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireManage(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var body reviewBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.absences.Reject(r.Context(), companyID, id, body.Notes); err != nil {
		writeErr(w, err)
		return
	}
	h.logAbsence(r.Context(), companyID, "rejected", id, nil)
	w.WriteHeader(http.StatusNoContent)
}

func mustTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id := tenant.GetCompanyID(r.Context())
	if id == uuid.Nil {
		http.Error(w, "empresa requerida", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	return id, true
}

// requireManage exige además rol owner/admin -- para aprobar/rechazar ausencias, de forma que
// un empleado no pueda resolver su propia solicitud (o la de un par) sin pasar por gestión.
func requireManage(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, ok := mustTenant(w, r)
	if !ok {
		return uuid.Nil, false
	}
	if !tenant.CanManage(r.Context()) {
		http.Error(w, "requiere rol de administrador", http.StatusForbidden)
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrAbsenceNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrAbsenceNotPending),
		errors.Is(err, domain.ErrInvalidDateRange):
		status = http.StatusBadRequest
	}
	http.Error(w, err.Error(), status)
}
