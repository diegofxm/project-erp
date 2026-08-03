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

func TestSchema_SalesTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"sales.sales", "id, company_id, customer_id, number, status, issue_date, due_date, notes, invoice_document_id, created_at, updated_at"},
		{"sales.sale_lines", "id, sale_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount"},
		{"sales.quotes", "id, company_id, customer_id, number, status, issue_date, valid_until, notes, created_at, updated_at"},
		{"sales.quote_lines", "id, quote_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount"},
		{"sales.sale_payments", "id, company_id, sale_id, payment_date, amount, payment_method, reference, notes, created_at"},
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
