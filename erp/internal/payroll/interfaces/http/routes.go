package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Empleados
	mux.HandleFunc("GET /api/v1/payroll/employees", h.handleListEmployees)
	mux.HandleFunc("POST /api/v1/payroll/employees", h.handleCreateEmployee)
	mux.HandleFunc("GET /api/v1/payroll/employees/{id}", h.handleGetEmployee)
	mux.HandleFunc("DELETE /api/v1/payroll/employees/{id}", h.handleDeactivateEmployee)

	// Contratos por empleado
	mux.HandleFunc("GET /api/v1/payroll/employees/{employee_id}/contracts", h.handleListContracts)
	mux.HandleFunc("POST /api/v1/payroll/contracts", h.handleCreateContract)
	mux.HandleFunc("DELETE /api/v1/payroll/contracts/{id}", h.handleTerminateContract)

	// Liquidaciones
	mux.HandleFunc("GET /api/v1/payroll/payslips", h.handleListPayslips)
	mux.HandleFunc("POST /api/v1/payroll/payslips", h.handleGeneratePayslip)
	mux.HandleFunc("GET /api/v1/payroll/payslips/{id}", h.handleGetPayslip)
	mux.HandleFunc("POST /api/v1/payroll/payslips/{id}/approve", h.handleApprovePayslip)
	mux.HandleFunc("POST /api/v1/payroll/payslips/{id}/pay", h.handleMarkPaidPayslip)
	mux.HandleFunc("POST /api/v1/payroll/payslips/{id}/void", h.handleVoidPayslip)
}
