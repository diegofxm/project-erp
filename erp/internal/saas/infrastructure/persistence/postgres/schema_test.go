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

func TestSchema_SaasTables(t *testing.T) {
	p := pool(t)
	tables := []struct {
		name string
		cols string
	}{
		{"saas.modules", "id, code, name, description, created_at"},
		{"saas.plans", `id, code, name, description, billing_cycle, price_cents,
			included_documents, price_per_extra_document_cents, requires_certificate,
			certificate_price_cents, annual_increment_pct, is_internal, is_active,
			created_at, updated_at`},
		{"saas.plan_modules", "plan_id, module_id"},
		{"saas.subscriptions", `id, company_id, plan_id, has_own_certificate, status,
			contracted_price_cents, current_period_start, current_period_end, cert_expires_at,
			created_at, updated_at`},
		{"saas.payments", "id, company_id, subscription_id, type, amount_cents, note, paid_at, created_at"},
		{"saas.settings", "id, iva_rate_bp, updated_at"},
		{"saas.prospects", `id, name, email, nit, cedula_file, cedula_content_type, rut_file,
			rut_content_type, status, notes, reviewed_at, created_at, updated_at`},
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
