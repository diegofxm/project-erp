//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestAccountingSchema(t *testing.T) {
	_ = godotenv.Load("../../../../../../.env")
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurado")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	tables := []string{
		"SELECT id, code, name, parent_code, level, category, is_posting, is_active, created_at, updated_at FROM accounting.accounts LIMIT 0",
		"SELECT id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at FROM accounting.accounting_periods LIMIT 0",
		"SELECT id, company_id, period_id, date, description, status, source, entry_type, voucher_type, voucher_number, source_document_id, source_document_type, book, created_at, updated_at FROM accounting.journal_entries LIMIT 0",
		"SELECT id, journal_id, account_id, debit, credit, cost_center, description, third_party_nit, foreign_amount, foreign_currency, created_at FROM accounting.journal_lines LIMIT 0",
		"SELECT id, company_id, code, name, resets_annually, is_active, created_at, updated_at FROM accounting.voucher_types LIMIT 0",
		"SELECT company_id, code, year, last_seq FROM accounting.voucher_counters LIMIT 0",
		"SELECT id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at FROM accounting.bank_accounts LIMIT 0",
		"SELECT id, bank_account_id, date, description, debit, credit, reference, is_reconciled, journal_line_id, created_at FROM accounting.bank_statement_lines LIMIT 0",
		"SELECT id, code, name, type, rate_bp, min_base_uvt, account_payable, account_receivable, applicable_to, is_active, created_at, updated_at FROM accounting.withholding_concepts LIMIT 0",
		"SELECT year, value_cents FROM accounting.uvt_values LIMIT 0",
		"SELECT id, company_id, code, name, description, asset_account, depreciation_account, accumulated_account, gain_account, loss_account, acquisition_date, acquisition_cost, salvage_value, useful_life_months, depreciation_method, status, third_party_nit, source_document_id, source_document_type, created_at, updated_at FROM accounting.fixed_assets LIMIT 0",
		"SELECT id, company_id, period_id, run_date, status, journal_id, created_at FROM accounting.depreciation_runs LIMIT 0",
		"SELECT id, run_id, asset_id, amount, created_at FROM accounting.depreciation_entries LIMIT 0",
		"SELECT id, company_id, period_start, period_end, period_type, generated_iva, deductible_iva, withheld_iva, net_iva, previous_balance, amount_to_pay, carry_forward, status, journal_id, filed_at, created_at, updated_at FROM accounting.iva_declarations LIMIT 0",
		"SELECT id, company_id, journal_line_id, reconciled_with, note, reconciled_at FROM accounting.reconciliation_marks LIMIT 0",
		"SELECT id, rate_date, from_currency, to_currency, rate_x10000, source, created_at FROM accounting.exchange_rates LIMIT 0",
		"SELECT id, company_id, year, name, status, created_at, updated_at FROM accounting.budgets LIMIT 0",
		"SELECT id, budget_id, account_id, jan, feb, mar, apr, may, jun, jul, aug, sep, oct, nov, dec, created_at, updated_at FROM accounting.budget_lines LIMIT 0",
		"SELECT year, rate_bp, created_at FROM accounting.income_tax_rates LIMIT 0",
		"SELECT id, company_id, fiscal_year, taxable_income, tax_rate_bp, tax_computed, discounts, tax_to_pay, advance_payments, amount_due, carry_forward, status, journal_id, filed_at, created_at, updated_at FROM accounting.income_tax_declarations LIMIT 0",
		"SELECT id, company_id, fiscal_year, third_party_nit, concept_code, concept_name, wh_type, gross_amount, tax_withheld, status, issued_at, created_at, updated_at FROM accounting.withholding_certificates LIMIT 0",
		"SELECT id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp, created_at, updated_at FROM accounting.ica_tariffs LIMIT 0",
		"SELECT id, company_id, municipality_code, period_start, period_end, period_type, ciiu_code, gross_revenue, deductions, net_base, tariff_bp, surcharge_bp, tax_computed, surcharge_amount, tax_to_pay, previous_balance, amount_due, carry_forward, status, journal_id, filed_at, created_at, updated_at FROM accounting.ica_declarations LIMIT 0",
	}

	for _, q := range tables {
		t.Run(q[:60], func(t *testing.T) {
			if _, err := pool.Exec(context.Background(), q); err != nil {
				t.Errorf("query falló: %v\nSQL: %s", err, q)
			}
		})
	}
}
