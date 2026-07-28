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

func TestSchema_CatalogTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"catalog.currencies", "code, name, symbol, created_at"},
		{"catalog.departments", "code, name, description, created_at"},
		{"catalog.municipalities", "code, name, department_code, description, created_at"},
		{"catalog.identification_types", "code, name, description, created_at"},
		{"catalog.payment_methods", "code, name, description, created_at"},
		{"catalog.payment_terms", "code, name, description, created_at"},
		{"catalog.dian_tax_types", "code, name, description, created_at"},
		{"catalog.tax_regimes", "code, name, description, created_at"},
		{"catalog.liability_codes", "code, name, description, created_at"},
		{"catalog.unit_measures", "code, name, description, created_at"},
		{"catalog.dian_document_types", "code, name, description, created_at"},
		{"catalog.ciiu_codes", "code, description, created_at, updated_at"},
		{"catalog.item_standards", "code, name, agency_id, description, created_at"},
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
