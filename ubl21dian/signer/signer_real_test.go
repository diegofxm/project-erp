package signer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/diegofxm/ubl21dian/builder"
	"github.com/diegofxm/ubl21dian/cufe"
	"github.com/diegofxm/ubl21dian/domain"
	"github.com/diegofxm/ubl21dian/qr"
	"github.com/diegofxm/ubl21dian/securitycode"
)

// parseEnvFile lee un archivo estilo .env (KEY=VALUE, líneas # son comentarios) sin
// depender de ninguna librería — solo se usa en este test, nunca en código de producción.
func parseEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer %s: %v", path, err)
	}
	vals := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		vals[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return vals
}

// TestSign_RealCertificate firma una factura real (datos del RUT 6382356) con el
// certificado y la clave técnica reales del ambiente de habilitación. Se omite por defecto
// — solo corre si UBL21DIAN_TEST_FIXTURES_DIR apunta a una carpeta con certificado_cert.pem,
// certificado_key.pem y credenciales.txt (mismo patrón que DATABASE_URL en core-bank: el
// material real nunca se commitea, así que la prueba se salta para cualquier otra persona).
func TestSign_RealCertificate(t *testing.T) {
	dir := os.Getenv("UBL21DIAN_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("UBL21DIAN_TEST_FIXTURES_DIR no configurado, se omite la prueba con credenciales reales")
	}

	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("leer certificado: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("leer llave privada: %v", err)
	}
	cert, key, err := LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}

	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	rangeFrom, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_FROM: %v", err)
	}
	rangeTo, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_TO: %v", err)
	}

	now := time.Now().In(domain.Bogota)
	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1: Factura Electrónica de Venta",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10",
		DocumentTypeCode:  "01",
		HashType:          "CUFE-SHA384",

		Prefix: env["DIAN_PREFIX"],
		Number: "1",

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

		// Datos reales del RUT 6382356 (Diego Fernando Montoya Vallejo, persona natural,
		// sin responsabilidad de IVA — mismo patrón que el caso persona natural verificado
		// en la factura real FESG27: TaxSchemeCode "ZZ"/"No aplica", TaxLevelCode "R-99-PN").
		Supplier: domain.Party{
			EntityTypeCode: "1",
			Identification: domain.Identification{Number: "6382356", TypeCode: "13"},
			Name:           "DIEGO FERNANDO MONTOYA VALLEJO",
			Address: domain.Address{
				Line:        "CL 13 A 25 26 BRR LAS AMERICAS",
				CityCode:    "76520",
				CityName:    "Palmira",
				StateCode:   "76",
				StateName:   "Valle del Cauca",
				CountryCode: "CO",
				CountryName: "Colombia",
			},
			LiabilityCodes: []string{"R-99-PN"},
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
			Phone:          "3186708084",
			Email:          "diegofm.comercial@gmail.com",
		},

		Customer: domain.Party{
			EntityTypeCode: "2",
			Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
			Name:           "Consumidor Final",
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
		},

		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},

		Totals: domain.Totals{
			LineExtensionCents: 10000,
			TaxExclusiveCents:  10000,
			TaxInclusiveCents:  10000,
			PayableCents:       10000,
		},

		Lines: []domain.Line{{
			Description:        "Servicio de prueba (TestSign_RealCertificate)",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			ItemCode:           "0001",
			ItemTypeCode:       "999",
			ItemTypeName:       "Estándar de adopción del contribuyente",
		}},

		NumberingRange: domain.NumberingRange{
			AuthorizedCode: env["DIAN_RESOLUTION"],
			Prefix:         env["DIAN_PREFIX"],
			StartNumber:    env["DIAN_RANGE_FROM"],
			EndNumber:      env["DIAN_RANGE_TO"],
			StartDate:      rangeFrom.Format("2006-01-02"),
			EndDate:        rangeTo.Format("2006-01-02"),
		},

		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "13"},
			SoftwareID:             env["DIAN_SOFTWARE_ID"],
		},
	}

	inv.CUFE = cufe.Compute(inv, env["DIAN_TECHNICAL_KEY"])
	inv.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	doc, err := builder.BuildInvoice(inv)
	if err != nil {
		t.Fatalf("BuildInvoice: %v", err)
	}

	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		t.Fatalf("SignaturePlaceholder: %v", err)
	}

	s := New(cert, key)
	if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	verifySignature(t, doc.Root(), &key.PublicKey)

	// Sin doc.Indent(): reescribiría el árbol después de firmar, dejando el archivo
	// guardado distinto de lo que realmente se canonicalizó/firmó (ver nota en
	// soap/realsend_test.go, donde esto causó un rechazo real de la DIAN).
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("WriteToString: %v", err)
	}
	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("crear outputs/: %v", err)
	}
	outPath := filepath.Join(outputsDir, "_signed_test_output.xml")
	if err := os.WriteFile(outPath, []byte(out), 0o644); err != nil {
		t.Fatalf("escribir salida: %v", err)
	}
	t.Logf("XML firmado escrito en %s (CUFE=%s)", outPath, inv.CUFE)
}
