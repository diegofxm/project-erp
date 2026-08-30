package integrationtest

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
)

// parseEnvFile reads a .env-style file (KEY=VALUE, lines starting with # are comments)
// without depending on any library — it is used only in this test, never in production code.
func parseEnvFile(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
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

// TestSendTestSetAsync_Real builds, signs, compresses, and sends a real invoice to the DIAN
// certification environment, using the real credentials from docs/reference. Skipped by
// default (same pattern as core-bank's DATABASE_URL) — it only runs if
// COFACTURE_TEST_FIXTURES_DIR points to that folder.
//
// This does NOT complete the certification process by itself — that requires submitting the
// complete "test set" that the DIAN defines for each participant, not a single made-up invoice.
// The goal here is to validate that our own pipeline (build -> CUFE -> sign -> zip -> submit)
// is accepted by the real server, not to finish certification in one shot.
func TestSendTestSetAsync_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping real-DIAN test")
	}

	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("read certificate: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}

	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	rangeFrom, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_DATE_FROM: %v", err)
	}
	rangeTo, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_DATE_TO: %v", err)
	}

	now := time.Now().In(domain.Bogota)
	// The number must fall inside the authorized range (DIAN_RANGE_FROM..DIAN_RANGE_TO) — the
	// DIAN rejected the first run of this test with FAD05b when using "1". A small time-based
	// offset is added so this test can be re-run several times without repeating the same
	// invoice number.
	rangeFromInt, err := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_FROM: %v", err)
	}
	number := strconv.FormatInt(rangeFromInt+time.Now().Unix()%100000, 10)

	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1: Factura Electrónica de Venta",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10",
		DocumentTypeCode:  "01",
		HashType:          "CUFE-SHA384",

		Prefix: env["DIAN_PREFIX"],
		Number: number,

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

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
			LiabilityCodes: []string{"R-99-PN"},
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
			Email:          "consumidor.final@example.com",
		},

		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},

		// HeaderTaxes with 0% VAT (instead of omitting TaxTotal entirely) — the DIAN rejected
		// the first run of this test with "FAU04: Base Imponible es distinto a la suma de los
		// valores de las bases imponibles de todas líneas de detalle" ("Taxable amount differs
		// from the sum of the taxable amounts of all detail lines") when no TaxTotal was reported
		// at all.
		HeaderTaxes: []domain.Tax{
			{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
		},

		Totals: domain.Totals{
			LineExtensionCents: 10000,
			TaxExclusiveCents:  10000,
			TaxInclusiveCents:  10000,
			PayableCents:       10000,
		},

		Lines: []domain.Line{{
			Description:        "Servicio de prueba (TestSendTestSetAsync_Real)",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			ItemCode:           "0001",
			ItemTypeCode:       "999",
			ItemTypeName:       "Estándar de adopción del contribuyente",
			Taxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
		}},

		NumberingRange: domain.NumberingRange{
			AuthorizedCode: env["DIAN_RESOLUTION"],
			Prefix:         env["DIAN_PREFIX"],
			StartNumber:    env["DIAN_RANGE_FROM"],
			EndNumber:      env["DIAN_RANGE_TO"],
			StartDate:      rangeFrom.Format("2006-01-02"),
			EndDate:        rangeTo.Format("2006-01-02"),
		},

		// The DIAN requires schemeName="31" (NIT) for the technology provider's ProviderID
		// even when, as here, the issuer is a natural person identified by cédula elsewhere in
		// the document (rule FAB23) — and the DV must be correctly computed too (rule FAB22b),
		// which is why it's "7" and not an arbitrary value.
		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "31", VerificationCode: "7"},
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
	s := signer.New(cert, key)
	if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// WARNING: never call doc.Indent() (or anything else that rewrites the tree) after signing —
	// that inserts whitespace text into the structure, and the document that gets transmitted
	// is no longer the same one that was canonicalized/signed. That was exactly the cause of a
	// real "Valor de la firma inválido" ("Invalid signature value") rejection from the DIAN in
	// an earlier run of this test.
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("WriteToBytes: %v", err)
	}
	// Save a local copy of the signed XML being sent, so it can be inspected separately if the
	// DIAN rejects it (docs/reference/outputs is in .gitignore, it isn't committed).
	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("create outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_send_testset_invoice.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("save local copy of the XML: %v", err)
	}

	fileName := zip.DocumentFileName(zip.KindInvoice, inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)

	client := soap.New(soap.HabilitacionURL, cert, key)
	result, err := client.SendTestSetAsync(zipFileName, zipBytes, env["DIAN_TEST_SET_ID"])
	if err != nil {
		t.Fatalf("SendTestSetAsync: %v", err)
	}

	t.Logf("CUFE: %s", inv.CUFE)
	t.Logf("ZipKey: %q", result.ZipKey)
	if result.ErrorMessageList != nil {
		for _, e := range result.ErrorMessageList.Items {
			t.Logf("Initial error: file=%s message=%s success=%v", e.XmlFileName, e.ProcessedMessage, e.Success)
		}
	}

	if result.ZipKey == "" {
		t.Fatal("DIAN did not return a ZipKey — check ErrorMessageList above")
	}

	// Validation is asynchronous; we retry GetStatusZip a few times before giving up on
	// waiting for the result within this same test run.
	for i := 0; i < 6; i++ {
		time.Sleep(5 * time.Second)
		statuses, err := client.GetStatusZip(result.ZipKey)
		if err != nil {
			t.Logf("GetStatusZip attempt %d: %v", i+1, err)
			continue
		}
		if len(statuses) == 0 {
			t.Logf("GetStatusZip attempt %d: no result yet", i+1)
			continue
		}
		for _, st := range statuses {
			t.Logf("Result: IsValid=%v StatusCode=%s StatusMessage=%s StatusDescription=%s",
				st.IsValid, st.StatusCode, st.StatusMessage, st.StatusDescription)
			if st.ErrorMessage != nil {
				for _, m := range st.ErrorMessage.Items {
					t.Logf("  ErrorMessage: %s", m)
				}
			}
		}
		return
	}
	t.Log("no result from GetStatusZip within this test's wait time; the ZipKey above can be queried later")
}
