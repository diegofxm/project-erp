//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func TestElectronicSchema(t *testing.T) {
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
		"SELECT id, company_id, dian_document_type_code, prefix, resolution_number, resolution_date, range_from, range_to, current_number, valid_from, valid_to, environment, technical_key, test_set_id, is_active, created_at, updated_at FROM electronic.numbering_ranges LIMIT 0",
		"SELECT id, company_id, numbering_range_id, dian_document_type_code, prefix, number, document_key, issue_date, issue_time, currency_code, customer, customer_id, lines, payment_means, totals_line_extension_cents, totals_tax_exclusive_cents, totals_tax_inclusive_cents, totals_prepaid_cents, totals_payable_cents, billing_reference, discrepancy_response, note_type_code, note, qr_url, signed_xml, status, dian_track_id, dian_status_code, dian_status_description, dian_status_message, application_response_xml, vendor, operation_type_code, withholding_taxes, vendor_id, created_at, updated_at FROM electronic.documents LIMIT 0",
	}
	for _, q := range tables {
		t.Run(q[:60], func(t *testing.T) {
			if _, err := pool.Exec(context.Background(), q); err != nil {
				t.Errorf("query falló: %v\nSQL: %s", err, q)
			}
		})
	}
}
