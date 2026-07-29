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

func TestSchema_CompanyTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"company.companies", `id, nit, check_digit, business_name, trade_name,
			identification_type_code, department_code, municipality_code, address_line,
			email, phone, environment, entity_type_code,
			tax_scheme_code, tax_scheme_name, liability_codes, tax_regime_code,
			industry_classification_codes, merchant_registration_number,
			software_id, software_pin_enc, certificate, certificate_password_enc,
			certificate_subject, certificate_issuer_cn, certificate_expires_at,
			ne_software_id, ne_software_pin_enc,
			logo, logo_content_type, is_active, created_at, updated_at`},
		{"company.warehouses", "id, company_id, code, name, address, is_active, created_at, updated_at"},
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
