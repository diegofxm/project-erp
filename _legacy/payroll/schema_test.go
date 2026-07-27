//go:build integration

package payroll_test

// TestPayrollSchemaMatchesCode verifica que las 2 migraciones del módulo de nómina
// generan exactamente las columnas que usa el código Go. Ejecutar con:
//
//	DATABASE_URL="postgres://..." go test -tags integration -v ./payroll/

import (
	"context"
	"fmt"
	"os"
	"testing"

	payroll "github.com/diegofxm/payroll"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPayrollSchemaMatchesCode(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurada — omitiendo test de integración de esquema")
	}

	ctx := context.Background()

	if err := payroll.Migrate(url); err != nil {
		t.Fatalf("migrar payroll: %v", err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("conectar BD: %v", err)
	}
	defer pool.Close()

	tables := []struct {
		name  string
		query string
	}{
		// ── 000001_employees ─────────────────────────────────────────────────────────────────
		{
			name: "payroll.smmlv_values",
			// mirrors database/seed/seed.go : SMMLV INSERT
			query: `SELECT year, amount_cents FROM payroll.smmlv_values LIMIT 0`,
		},
		{
			name: "payroll.arl_rates",
			// mirrors database/seed/seed.go : ARLRates INSERT
			query: `SELECT year, risk_class, rate_bp FROM payroll.arl_rates LIMIT 0`,
		},
		{
			name: "payroll.employees",
			// mirrors employees/postgres.go : employeeCols
			query: `SELECT id, company_id, identification_type_code, identification_number,
			        first_name, last_name, email, phone,
			        department_code, municipality_code, address_line,
			        is_active, created_at, updated_at
			        FROM payroll.employees LIMIT 0`,
		},
		{
			name: "payroll.contracts",
			// mirrors contracts/postgres.go : contractCols
			query: `SELECT id, employee_id, company_id,
			        contract_type, work_schedule, position, cost_center,
			        salary_cents, salary_type, risk_class,
			        start_date, end_date, termination_date, termination_cause,
			        health_entity, pension_entity, arl_entity, caja_entity,
			        is_active, created_at, updated_at
			        FROM payroll.contracts LIMIT 0`,
		},
		// ── 000002_payslips ──────────────────────────────────────────────────────────────────
		{
			name: "payroll.payslips",
			// mirrors payslips/postgres.go : payslipCols
			query: `SELECT id, company_id, employee_id, contract_id,
			        period_year, period_month, worked_days, status,
			        total_earned_cents, total_deducted_cents, net_pay_cents,
			        journal_id, paid_at, created_at, updated_at
			        FROM payroll.payslips LIMIT 0`,
		},
		{
			name: "payroll.payslip_lines",
			// mirrors payslips/postgres.go : lineCols
			query: `SELECT id, payslip_id, concept_code, concept_name, concept_type,
			        quantity, amount_cents, created_at
			        FROM payroll.payslip_lines LIMIT 0`,
		},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, tt.query); err != nil {
				t.Errorf("esquema no coincide con el código: %v", err)
			}
		})
	}
}

// TestPayrollSeedLoads verifica que las funciones de seed no retornan error
// y que las tablas de catálogo quedan con datos.
func TestPayrollSeedLoads(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurada")
	}

	ctx := context.Background()

	if err := payroll.Migrate(url); err != nil {
		t.Fatalf("migrar: %v", err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("conectar BD: %v", err)
	}
	defer pool.Close()

	core := payroll.New(pool)
	if err := core.Seed(ctx); err != nil {
		t.Fatalf("seed falló: %v", err)
	}

	catalogs := []struct{ table, pk string }{
		{"payroll.smmlv_values", "year"},
		{"payroll.arl_rates", "year"},
	}
	for _, c := range catalogs {
		t.Run(c.table, func(t *testing.T) {
			var count int
			if err := pool.QueryRow(ctx,
				fmt.Sprintf("SELECT COUNT(*) FROM %s", c.table),
			).Scan(&count); err != nil {
				t.Fatalf("contar filas: %v", err)
			}
			if count == 0 {
				t.Errorf("%s quedó vacía después del seed", c.table)
			}
		})
	}
}
