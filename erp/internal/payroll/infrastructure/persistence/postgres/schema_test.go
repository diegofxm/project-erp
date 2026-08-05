//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no definida")
	}
	p, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("abrir pool: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestSchema_PayrollTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"payroll.smmlv_values", "year, amount_cents"},
		{"payroll.arl_rates", "year, risk_class, rate_bp"},
		{"payroll.employees", `id, company_id, identification_type_code, identification_number,
			first_name, last_name, email, phone,
			department_code, municipality_code, address_line,
			is_active, created_at, updated_at`},
		{"payroll.contracts", `id, employee_id, company_id,
			contract_type, work_schedule, position, cost_center,
			salary_cents, salary_type, risk_class,
			start_date, end_date, termination_date, termination_cause,
			health_entity, pension_entity, arl_entity, caja_entity,
			is_active, created_at, updated_at`},
		{"payroll.payslips", `id, company_id, number, employee_id, contract_id,
			period_year, period_month, worked_days, status,
			total_earned_cents, total_deducted_cents, net_pay_cents,
			journal_id, paid_at, created_at, updated_at`},
		{"payroll.payslip_lines", `id, payslip_id, concept_code, concept_name, concept_type,
			quantity, amount_cents, created_at`},
		{"payroll.number_counters", "company_id, year, last_seq"},
	}
	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Exec(context.Background(),
				"SELECT "+tt.cols+" FROM "+tt.name+" LIMIT 0")
			if err != nil {
				t.Fatalf("tabla %s: %v", tt.name, err)
			}
		})
	}
}
