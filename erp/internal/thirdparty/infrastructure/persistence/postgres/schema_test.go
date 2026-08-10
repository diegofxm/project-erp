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

func TestSchema_ThirdpartyTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"thirdparty.parties", `id, company_id,
			identification_type_code, identification_number, check_digit,
			entity_type_code, merchant_registration_number,
			name, tax_scheme_code, tax_scheme_name, tax_regime_code, liability_codes,
			department_code, municipality_code, address_line,
			address_city_name, address_state_name, address_country_code, address_country_name,
			email, phone,
			is_customer, is_supplier, credit_limit, payment_terms_days,
			habeas_data_consent, habeas_data_consent_at,
			is_active, created_at, updated_at`},
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
