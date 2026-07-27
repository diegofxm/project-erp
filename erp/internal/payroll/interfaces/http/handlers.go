package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/payroll/application"
	"github.com/diegofxm/erp/internal/payroll/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	employees application.EmployeeUseCase
	contracts application.ContractUseCase
	payslips  application.PayslipUseCase
}

func NewHandler(
	employees *application.EmployeeUseCase,
	contracts *application.ContractUseCase,
	payslips *application.PayslipUseCase,
) *Handler {
	return &Handler{
		employees: *employees,
		contracts: *contracts,
		payslips:  *payslips,
	}
}

// ── Empleados ─────────────────────────────────────────────────────────────────

type createEmployeeBody struct {
	IdentificationTypeCode string `json:"identification_type_code"`
	IdentificationNumber   string `json:"identification_number"`
	FirstName              string `json:"first_name"`
	LastName               string `json:"last_name"`
	Email                  string `json:"email"`
	Phone                  string `json:"phone"`
	DepartmentCode         string `json:"department_code"`
	MunicipalityCode       string `json:"municipality_code"`
	AddressLine            string `json:"address_line"`
}

func (h *Handler) handleCreateEmployee(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body createEmployeeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	e, err := h.employees.Create(r.Context(), domain.CreateEmployeeInput{
		CompanyID:              companyID,
		IdentificationTypeCode: body.IdentificationTypeCode,
		IdentificationNumber:   body.IdentificationNumber,
		FirstName:              body.FirstName,
		LastName:               body.LastName,
		Email:                  body.Email,
		Phone:                  body.Phone,
		DepartmentCode:         body.DepartmentCode,
		MunicipalityCode:       body.MunicipalityCode,
		AddressLine:            body.AddressLine,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) handleListEmployees(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	list, err := h.employees.List(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) handleGetEmployee(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	e, err := h.employees.Get(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) handleDeactivateEmployee(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.employees.Deactivate(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Contratos ─────────────────────────────────────────────────────────────────

type createContractBody struct {
	EmployeeID    uuid.UUID          `json:"employee_id"`
	ContractType  domain.ContractType  `json:"contract_type"`
	WorkSchedule  string             `json:"work_schedule"`
	Position      string             `json:"position"`
	CostCenter    string             `json:"cost_center"`
	SalaryCents   int64              `json:"salary_cents"`
	SalaryType    domain.SalaryType    `json:"salary_type"`
	RiskClass     domain.WorkRiskClass `json:"risk_class"`
	StartDate     string             `json:"start_date"` // YYYY-MM-DD
	EndDate       string             `json:"end_date"`
	HealthEntity  string             `json:"health_entity"`
	PensionEntity string             `json:"pension_entity"`
	ARLEntity     string             `json:"arl_entity"`
	CajaEntity    string             `json:"caja_entity"`
}

func (h *Handler) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body createContractBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in := domain.CreateContractInput{
		EmployeeID:    body.EmployeeID,
		ContractType:  body.ContractType,
		WorkSchedule:  body.WorkSchedule,
		Position:      body.Position,
		CostCenter:    body.CostCenter,
		SalaryCents:   body.SalaryCents,
		SalaryType:    body.SalaryType,
		RiskClass:     body.RiskClass,
		HealthEntity:  body.HealthEntity,
		PensionEntity: body.PensionEntity,
		ARLEntity:     body.ARLEntity,
		CajaEntity:    body.CajaEntity,
	}
	if body.StartDate != "" {
		if t, err := time.Parse("2006-01-02", body.StartDate); err == nil {
			in.StartDate = t
		}
	}
	if body.EndDate != "" {
		if t, err := time.Parse("2006-01-02", body.EndDate); err == nil {
			in.EndDate = &t
		}
	}
	if in.WorkSchedule == "" {
		in.WorkSchedule = "full_time"
	}
	c, err := h.contracts.Create(r.Context(), companyID, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) handleListContracts(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	empID, err := uuid.Parse(r.PathValue("employee_id"))
	if err != nil {
		http.Error(w, "employee_id inválido", http.StatusBadRequest)
		return
	}
	list, err := h.contracts.ListByEmployee(r.Context(), companyID, empID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type terminateBody struct {
	Cause string `json:"cause"`
}

func (h *Handler) handleTerminateContract(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	var body terminateBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := h.contracts.Terminate(r.Context(), companyID, id, body.Cause); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Liquidaciones ─────────────────────────────────────────────────────────────

type generatePayslipBody struct {
	EmployeeID  uuid.UUID `json:"employee_id"`
	ContractID  uuid.UUID `json:"contract_id"`
	PeriodYear  int       `json:"period_year"`
	PeriodMonth int       `json:"period_month"`
	WorkedDays  int       `json:"worked_days"`
}

func (h *Handler) handleGeneratePayslip(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	var body generatePayslipBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ps, err := h.payslips.Generate(r.Context(), companyID, domain.CreatePayslipInput{
		EmployeeID:  body.EmployeeID,
		ContractID:  body.ContractID,
		PeriodYear:  body.PeriodYear,
		PeriodMonth: body.PeriodMonth,
		WorkedDays:  body.WorkedDays,
	})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ps)
}

func (h *Handler) handleGetPayslip(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	ps, err := h.payslips.Get(r.Context(), companyID, id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ps)
}

func (h *Handler) handleListPayslips(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	list, err := h.payslips.List(r.Context(), companyID, year, month)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) handleApprovePayslip(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.payslips.Approve(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleMarkPaidPayslip(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.payslips.MarkPaid(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleVoidPayslip(w http.ResponseWriter, r *http.Request) {
	companyID, ok := mustTenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.payslips.Void(r.Context(), companyID, id); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id := tenant.GetCompanyID(r.Context())
	if id == uuid.Nil {
		http.Error(w, "empresa requerida", http.StatusUnauthorized)
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
	case errors.Is(err, domain.ErrEmployeeNotFound),
		errors.Is(err, domain.ErrContractNotFound),
		errors.Is(err, domain.ErrPayslipNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrEmployeeExists),
		errors.Is(err, domain.ErrPayslipExists),
		errors.Is(err, domain.ErrPayslipNotDraft),
		errors.Is(err, domain.ErrPayslipNotApproved),
		errors.Is(err, domain.ErrNoActiveContract),
		errors.Is(err, domain.ErrNoSMMLV),
		errors.Is(err, domain.ErrNoARLRate):
		status = http.StatusBadRequest
	}
	http.Error(w, err.Error(), status)
}
