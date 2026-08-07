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

func TestSchema_AccountingTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"accounting.accounts", "id, code, name, parent_code, level, category, is_posting, is_active, created_at, updated_at"},
		{"accounting.accounting_periods", "id, company_id, year, month, status, opened_at, closed_at, created_at, updated_at"},
		{"accounting.journal_entries", `id, company_id, period_id, date, description, status,
			source, entry_type, voucher_type, voucher_number,
			source_document_id, source_document_type, book, created_at, updated_at`},
		{"accounting.journal_lines", "id, journal_id, account_id, debit, credit, cost_center, description, third_party_nit, foreign_amount, foreign_currency, created_at"},
		{"accounting.voucher_types", "id, company_id, code, name, resets_annually, is_active, created_at, updated_at"},
		{"accounting.voucher_counters", "company_id, code, year, last_seq"},
		{"accounting.bank_accounts", "id, company_id, name, bank_name, account_no, account_id, is_active, created_at, updated_at"},
		{"accounting.bank_statement_lines", "id, bank_account_id, date, description, debit, credit, reference, is_reconciled, journal_line_id, created_at"},
		{"accounting.withholding_concepts", "id, code, name, type, rate_bp, min_base_uvt, account_payable, account_receivable, applicable_to, is_active, created_at, updated_at"},
		{"accounting.uvt_values", "year, value_cents"},
		{"accounting.fixed_assets", `id, company_id, code, name, description,
			asset_account, depreciation_account, accumulated_account, gain_account, loss_account,
			acquisition_date, acquisition_cost, salvage_value, useful_life_months, depreciation_method,
			status, third_party_nit, source_document_id, source_document_type, created_at, updated_at`},
		{"accounting.depreciation_runs", "id, company_id, period_id, run_date, status, journal_id, created_at"},
		{"accounting.depreciation_entries", "id, run_id, asset_id, amount, created_at"},
		{"accounting.iva_declarations", `id, company_id, period_start, period_end, period_type,
			generated_iva, deductible_iva, withheld_iva, net_iva,
			previous_balance, amount_to_pay, carry_forward,
			status, journal_id, filed_at, created_at, updated_at`},
		{"accounting.reconciliation_marks", "id, company_id, journal_line_id, reconciled_with, note, reconciled_at"},
		{"accounting.exchange_rates", "id, rate_date, from_currency, to_currency, rate_x10000, source, description, created_at"},
		{"accounting.budgets", "id, company_id, year, name, status, created_at, updated_at"},
		{"accounting.budget_lines", "id, budget_id, account_id, jan, feb, mar, apr, may, jun, jul, aug, sep, oct, nov, dec, created_at, updated_at"},
		{"accounting.income_tax_rates", "year, rate_bp, created_at"},
		{"accounting.income_tax_declarations", `id, company_id, fiscal_year, taxable_income,
			tax_rate_bp, tax_computed, discounts, tax_to_pay,
			advance_payments, amount_due, carry_forward,
			status, journal_id, filed_at, created_at, updated_at`},
		{"accounting.withholding_certificates", `id, company_id, number, fiscal_year, third_party_nit,
			concept_code, concept_name, wh_type,
			gross_amount, tax_withheld, status, issued_at, created_at, updated_at`},
		{"accounting.certificate_counters", "company_id, year, last_seq"},
		{"accounting.ica_tariffs", "id, municipality_code, ciiu_code, fiscal_year, rate_bp, surcharge_bp, created_at, updated_at"},
		{"accounting.ica_declarations", `id, company_id, municipality_code,
			period_start, period_end, period_type, ciiu_code,
			gross_revenue, deductions, net_base, tariff_bp, surcharge_bp,
			tax_computed, surcharge_amount, tax_to_pay,
			previous_balance, amount_due, carry_forward,
			status, journal_id, filed_at, created_at, updated_at`},
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
