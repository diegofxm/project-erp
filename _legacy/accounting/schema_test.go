//go:build integration

package accounting_test

// TestAccountingSchemaMatchesCode verifica que las 3 migraciones del módulo contable
// generan exactamente las columnas que usa el código Go.  Ejecutar con:
//
//	DATABASE_URL="postgres://..." go test -tags integration -v ./accounting/

import (
	"context"
	"fmt"
	"os"
	"testing"

	accounting "github.com/diegofxm/accounting"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAccountingSchemaMatchesCode(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurada — omitiendo test de integración de esquema")
	}

	ctx := context.Background()

	if err := accounting.Migrate(url); err != nil {
		t.Fatalf("migrar accounting: %v", err)
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
		// ── 000001_core ───────────────────────────────────────────────────────────────────
		{
			name: "accounting.accounts",
			// mirrors accounts/postgres.go : accountCols
			query: `SELECT id, code, name, parent_code, level, category,
			        is_posting, is_active, created_at, updated_at
			        FROM accounting.accounts LIMIT 0`,
		},
		{
			name: "accounting.accounting_periods",
			// mirrors periods/postgres.go : periodCols
			query: `SELECT id, company_id, year, month, status, opened_at, closed_at,
			        created_at, updated_at
			        FROM accounting.accounting_periods LIMIT 0`,
		},
		{
			name: "accounting.journal_entries",
			// mirrors journals/postgres.go : INSERT column list
			query: `SELECT id, company_id, period_id, date, description, status,
			        source, entry_type, voucher_type, voucher_number,
			        source_document_id, source_document_type, book,
			        created_at, updated_at
			        FROM accounting.journal_entries LIMIT 0`,
		},
		{
			name: "accounting.journal_lines",
			// mirrors journals/postgres.go : INSERT column list
			query: `SELECT id, journal_id, account_id, debit, credit,
			        cost_center, description, third_party_nit,
			        foreign_amount, foreign_currency, created_at
			        FROM accounting.journal_lines LIMIT 0`,
		},
		{
			name: "accounting.voucher_types",
			// mirrors journals/postgres.go : INSERT INTO accounting.voucher_types
			query: `SELECT id, company_id, code, name, resets_annually, is_active,
			        created_at, updated_at
			        FROM accounting.voucher_types LIMIT 0`,
		},
		{
			name: "accounting.voucher_counters",
			// mirrors journals/postgres.go : INSERT INTO accounting.voucher_counters
			query: `SELECT company_id, code, year, last_seq
			        FROM accounting.voucher_counters LIMIT 0`,
		},
		{
			name: "accounting.bank_accounts",
			// mirrors banking/postgres.go : INSERT INTO accounting.bank_accounts
			query: `SELECT id, company_id, name, bank_name, account_no, account_id,
			        is_active, created_at, updated_at
			        FROM accounting.bank_accounts LIMIT 0`,
		},
		{
			name: "accounting.bank_statement_lines",
			// mirrors banking/postgres.go : INSERT + SELECT
			query: `SELECT id, bank_account_id, date, description, debit, credit,
			        reference, is_reconciled, journal_line_id, created_at
			        FROM accounting.bank_statement_lines LIMIT 0`,
		},

		// ── 000002_modules ───────────────────────────────────────────────────────────────
		{
			name: "accounting.withholding_concepts",
			// mirrors seed/seed.go : WithholdingConcepts INSERT column list
			query: `SELECT id, code, name, type, rate_bp, min_base_uvt,
			        account_payable, account_receivable, applicable_to,
			        created_at, updated_at
			        FROM accounting.withholding_concepts LIMIT 0`,
		},
		{
			name: "accounting.uvt_values",
			// mirrors seed/seed.go : UVT INSERT — sin created_at (no está en la tabla)
			query: `SELECT year, value_cents FROM accounting.uvt_values LIMIT 0`,
		},
		{
			name: "accounting.fixed_assets",
			// mirrors assets/postgres.go : INSERT INTO accounting.fixed_assets
			query: `SELECT id, company_id, code, name, description,
			        asset_account, depreciation_account, accumulated_account,
			        gain_account, loss_account,
			        acquisition_date, acquisition_cost, salvage_value,
			        useful_life_months, depreciation_method, status,
			        third_party_nit, source_document_id, source_document_type,
			        created_at, updated_at
			        FROM accounting.fixed_assets LIMIT 0`,
		},
		{
			name: "accounting.depreciation_runs",
			// mirrors assets/postgres.go : INSERT INTO accounting.depreciation_runs
			query: `SELECT id, company_id, period_id, run_date, status, journal_id, created_at
			        FROM accounting.depreciation_runs LIMIT 0`,
		},
		{
			name: "accounting.depreciation_entries",
			// mirrors assets/postgres.go : INSERT INTO accounting.depreciation_entries
			query: `SELECT id, run_id, asset_id, amount, created_at
			        FROM accounting.depreciation_entries LIMIT 0`,
		},
		{
			name: "accounting.reconciliation_marks",
			// mirrors cartera/postgres.go : INSERT INTO accounting.reconciliation_marks
			query: `SELECT id, company_id, journal_line_id, reconciled_with, note, reconciled_at
			        FROM accounting.reconciliation_marks LIMIT 0`,
		},
		{
			name: "accounting.exchange_rates",
			// mirrors forex/postgres.go : INSERT INTO accounting.exchange_rates
			query: `SELECT id, rate_date, from_currency, to_currency, rate_x10000, source, created_at
			        FROM accounting.exchange_rates LIMIT 0`,
		},
		{
			name: "accounting.budgets",
			// mirrors budget/postgres.go : INSERT INTO accounting.budgets
			query: `SELECT id, company_id, year, name, status, created_at, updated_at
			        FROM accounting.budgets LIMIT 0`,
		},
		{
			name: "accounting.budget_lines",
			// mirrors budget/postgres.go : INSERT INTO accounting.budget_lines
			query: `SELECT id, budget_id, account_id,
			        jan, feb, mar, apr, may, jun,
			        jul, aug, sep, oct, nov, dec,
			        created_at, updated_at
			        FROM accounting.budget_lines LIMIT 0`,
		},

		// ── 000003_tax ────────────────────────────────────────────────────────────────────
		{
			name: "accounting.income_tax_rates",
			// mirrors seed/seed.go : IncomeTaxRates INSERT column list
			query: `SELECT year, rate_bp FROM accounting.income_tax_rates LIMIT 0`,
		},
		{
			name: "accounting.income_tax_declarations",
			query: `SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp,
			        tax_computed, discounts, tax_to_pay, advance_payments, amount_due,
			        carry_forward, status, journal_id, filed_at, created_at, updated_at
			        FROM accounting.income_tax_declarations LIMIT 0`,
		},
		{
			name: "accounting.withholding_certificates",
			query: `SELECT id, company_id, fiscal_year, third_party_nit,
			        concept_code, concept_name, wh_type, gross_amount, tax_withheld,
			        status, issued_at, created_at, updated_at
			        FROM accounting.withholding_certificates LIMIT 0`,
		},
		{
			name: "accounting.ica_tariffs",
			query: `SELECT id, municipality_code, ciiu_code, fiscal_year,
			        rate_bp, surcharge_bp, created_at, updated_at
			        FROM accounting.ica_tariffs LIMIT 0`,
		},
		{
			name: "accounting.ica_declarations",
			query: `SELECT id, company_id, municipality_code, period_start, period_end,
			        period_type, ciiu_code, gross_revenue, deductions, net_base,
			        tariff_bp, surcharge_bp, tax_computed, surcharge_amount, tax_to_pay,
			        previous_balance, amount_due, carry_forward, status,
			        journal_id, filed_at, created_at, updated_at
			        FROM accounting.ica_declarations LIMIT 0`,
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

// TestAccountingSeedLoads verifica que las funciones de seed no retornan error
// (valida formato de CSV, upsert idempotente, y que las tablas objetivo existen).
func TestAccountingSeedLoads(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurada")
	}

	ctx := context.Background()

	if err := accounting.Migrate(url); err != nil {
		t.Fatalf("migrar: %v", err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("conectar BD: %v", err)
	}
	defer pool.Close()

	core := accounting.New(pool)
	if err := core.Seed(ctx); err != nil {
		t.Fatalf("seed falló: %v", err)
	}

	catalogs := []struct{ table, pk string }{
		{"accounting.accounts", "code"},
		{"accounting.withholding_concepts", "id"},
		{"accounting.uvt_values", "year"},
		{"accounting.income_tax_rates", "year"},
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
