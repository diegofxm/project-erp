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

func TestSchema_PurchaseTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"purchase.orders", "id, company_id, supplier_id, number, status, issue_date, due_date, notes, payment_means, created_at, updated_at, support_document_id"},
		{"purchase.order_lines", "id, purchase_order_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount, unit_code"},
		{"purchase.purchase_payments", "id, company_id, purchase_id, payment_date, amount, payment_method, reference, notes, created_at"},
		{"purchase.purchase_withholdings", "id, purchase_order_id, concept_code, concept_name, base, rate_bp, amount, account_payable, created_at"},
		{"purchase.number_counters", "company_id, year, last_seq"},
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
