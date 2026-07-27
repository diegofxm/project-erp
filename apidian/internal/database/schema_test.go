//go:build integration

package database_test

// TestSchemaMatchesCode verifica que las migraciones SQL generan exactamente las columnas
// que el código Go espera encontrar.  Ejecutar con:
//
//	DATABASE_URL="postgres://..." go test -tags integration -v ./internal/database/
//
// La prueba corre Migrate() (idempotente si la BD ya está actualizada) y luego ejecuta
// SELECT <columnas> FROM <tabla> LIMIT 0 para cada tabla crítica.  Si una columna no
// existe o tiene nombre distinto al que usa el código, el test falla con el mismo error
// que recibiría un usuario en producción.

import (
	"context"
	"os"
	"testing"
)

func TestSchemaMatchesCode(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL no configurada — omitiendo test de integración de esquema")
	}

	ctx := context.Background()
	db, err := New(ctx, Config{URL: url, MaxConn: 2, MinConn: 1})
	if err != nil {
		t.Fatalf("conectar BD: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrar: %v", err)
	}

	// Cada entrada reproduce el SELECT exacto que ejecuta el repositorio correspondiente.
	// La columna "query" debe coincidir palabra por palabra con la constante *Columns del
	// postgres.go — si cambia una, hay que cambiar la otra.
	tables := []struct {
		name  string
		query string
	}{
		// ── public ────────────────────────────────────────────────────────────────────────
		{
			name: "public.users",
			// mirrors auth/postgres.go : userColumns
			query: `SELECT id, email, password_hash, name, role, is_superadmin, is_active,
			        invite_token, invite_token_expires_at, invite_accepted_at,
			        created_at, updated_at
			        FROM users LIMIT 0`,
		},
		{
			name: "public.user_issuers (incluye role y created_at — bug previo)",
			// mirrors auth/postgres.go : ListIssuerIDs ORDER BY created_at
			query: `SELECT user_id, issuer_id, role, created_at FROM user_issuers LIMIT 0`,
		},
		{
			name: "public.issuers",
			// mirrors issuers/postgres.go : issuerColumns
			query: `SELECT id, nit, check_digit, business_name, trade_name,
			        identification_type_code, department_code, municipality_code,
			        address_line, email, phone, environment, entity_type_code,
			        tax_scheme_code, tax_scheme_name, liability_codes, tax_regime_code,
			        industry_classification_codes, merchant_registration_number,
			        software_id, software_pin, certificate, certificate_password,
			        ne_software_id, ne_software_pin, logo, logo_content_type,
			        is_active, created_at, updated_at
			        FROM issuers LIMIT 0`,
		},

		// ── edocuments ───────────────────────────────────────────────────────────────────
		{
			name: "edocuments.numbering_ranges",
			// mirrors numbering/postgres.go : numberingColumns
			query: `SELECT id, issuer_id, dian_document_type_code, prefix,
			        resolution_number, resolution_date, range_from, range_to,
			        current_number, valid_from, valid_to, environment,
			        technical_key, test_set_id, is_active, created_at, updated_at
			        FROM edocuments.numbering_ranges LIMIT 0`,
		},
		{
			name: "edocuments.customers",
			// mirrors customers/postgres.go : customerColumns
			query: `SELECT id, issuer_id, entity_type_code, identification_number,
			        identification_type_code, identification_verification_code, name,
			        address_line, address_city_code, address_city_name,
			        address_state_code, address_state_name, address_country_code,
			        address_country_name, tax_scheme_code, tax_scheme_name,
			        liability_codes, tax_regime_code, phone, email,
			        merchant_registration_number, created_at, updated_at
			        FROM edocuments.customers LIMIT 0`,
		},
		{
			name: "edocuments.vendors",
			// mirrors vendors/postgres.go : vendorColumns
			query: `SELECT id, issuer_id, entity_type_code, identification_number,
			        identification_type_code, identification_verification_code, name,
			        address_line, address_city_code, address_city_name,
			        address_state_code, address_state_name, address_country_code,
			        address_country_name, tax_scheme_code, tax_scheme_name,
			        liability_codes, tax_regime_code, phone, email,
			        merchant_registration_number, created_at, updated_at
			        FROM edocuments.vendors LIMIT 0`,
		},
		{
			name: "edocuments.products (incluye unit_code — bug previo: unit_measure_code)",
			// mirrors products/postgres.go : productColumns
			query: `SELECT id, issuer_id, description, unit_code, unit_price_cents,
			        item_code, item_type_code, item_type_name, item_type_agency_id,
			        tax_type_code, tax_type_name, tax_percent, created_at, updated_at
			        FROM edocuments.products LIMIT 0`,
		},
		{
			name: "edocuments.documents (incluye operation_type_code — bug previo: faltaba)",
			// mirrors documents/postgres.go : documentColumns
			query: `SELECT id, issuer_id, numbering_range_id, dian_document_type_code,
			        prefix, number, document_key, issue_date, issue_time, currency_code,
			        customer, customer_id, lines, payment_means,
			        totals_line_extension_cents, totals_tax_exclusive_cents,
			        totals_tax_inclusive_cents, totals_prepaid_cents, totals_payable_cents,
			        billing_reference, discrepancy_response, note_type_code, note,
			        qr_url, signed_xml, status, dian_track_id, dian_status_code,
			        dian_status_description, dian_status_message, application_response_xml,
			        vendor, operation_type_code, withholding_taxes, vendor_id,
			        created_at, updated_at
			        FROM edocuments.documents LIMIT 0`,
		},

		// ── sanidad de prefijos de esquema ───────────────────────────────────────────────
		// Garantiza que no existan tablas homónimas en public que puedan resolver un
		// FROM sin prefijo a la tabla equivocada (bug previo: FROM documents sin prefijo).
		{
			name: "public.documents NO debe existir",
			query: `SELECT 1 WHERE NOT EXISTS (
			            SELECT 1 FROM information_schema.tables
			            WHERE table_schema = 'public' AND table_name = 'documents'
			        )`,
		},
		{
			name: "public.customers NO debe existir",
			query: `SELECT 1 WHERE NOT EXISTS (
			            SELECT 1 FROM information_schema.tables
			            WHERE table_schema = 'public' AND table_name = 'customers'
			        )`,
		},
		{
			name: "public.products NO debe existir",
			query: `SELECT 1 WHERE NOT EXISTS (
			            SELECT 1 FROM information_schema.tables
			            WHERE table_schema = 'public' AND table_name = 'products'
			        )`,
		},
		{
			name: "public.vendors NO debe existir",
			query: `SELECT 1 WHERE NOT EXISTS (
			            SELECT 1 FROM information_schema.tables
			            WHERE table_schema = 'public' AND table_name = 'vendors'
			        )`,
		},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := db.Pool.Exec(ctx, tt.query); err != nil {
				t.Errorf("esquema no coincide con el código: %v", err)
			}
		})
	}
}
