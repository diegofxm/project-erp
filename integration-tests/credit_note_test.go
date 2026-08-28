package integrationtest

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cude"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
)

// TestSendTestSetAsync_CreditNote_Real sends a real Credit Note referencing invoice
// SETP-990068706 (already authorized in an earlier run of this test) — it validates the CUDE
// computation for notes and the CreditNote structure against the real server, not just against
// the Technical Annex examples. Skipped by default, like the other tests that use real
// credentials.
func TestSendTestSetAsync_CreditNote_Real(t *testing.T) {
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

	now := time.Now().In(domain.Bogota)
	rangeFromInt, err := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
	if err != nil {
		t.Fatalf("parse DIAN_RANGE_FROM: %v", err)
	}
	// Same range as the invoice, but a different consecutive number so it doesn't collide with it.
	number := strconv.FormatInt(rangeFromInt+1+time.Now().Unix()%100000, 10)

	base := domain.Invoice{
		ProfileID:         "DIAN 2.1: Nota Crédito de Factura Electrónica de Venta",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "20", // credit note referencing a specific invoice
		DocumentTypeCode:  "91",
		HashType:          "CUDE-SHA384",

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
			Description:        "Anulación de servicio de prueba (TestSendTestSetAsync_CreditNote_Real)",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			Taxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 0, Percent: 0, TypeCode: "01", TypeName: "IVA"},
			},
		}},

		NumberingRange: domain.NumberingRange{
			AuthorizedCode: env["DIAN_RESOLUTION"],
			Prefix:         env["DIAN_PREFIX"],
			StartNumber:    env["DIAN_RANGE_FROM"],
			EndNumber:      env["DIAN_RANGE_TO"],
		},

		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "31", VerificationCode: "7"},
			SoftwareID:             env["DIAN_SOFTWARE_ID"],
		},
	}

	cn := domain.CreditNote{
		Invoice:            base,
		CreditNoteTypeCode: "91", // fixed DIAN code for Credit Notes; the reason code (2 = cancellation) goes in DiscrepancyResponse.ResponseCode
		BillingReference: domain.BillingReference{
			Prefix:    env["DIAN_PREFIX"],
			Number:    "990068706", // the real invoice authorized in TestSendTestSetAsync_Real
			CUFE:      "853657dcf2841c55c04338b24cc4db9dfbf87042f1ce1798e53f7b1f0502d00df9bd3f371dea47b02766424976d60ba2",
			IssueDate: "2026-06-20",
		},
		DiscrepancyResponse: &domain.DiscrepancyResponse{
			ReferenceID:  env["DIAN_PREFIX"] + "990068706",
			ResponseCode: "2",
			Description:  "Anulación de factura electrónica",
		},
	}

	cn.CUFE = cude.Compute(cn.Invoice, env["DIAN_PIN"])
	cn.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], cn.Prefix+cn.Number)
	cn.QRURL = qr.URL(cn.EnvironmentCode, cn.CUFE)

	doc, err := builder.BuildCreditNote(cn)
	if err != nil {
		t.Fatalf("BuildCreditNote: %v", err)
	}
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		t.Fatalf("SignaturePlaceholder: %v", err)
	}
	s := signer.New(cert, key)
	if err := s.Sign(doc.Root(), placeholder, "supplier", now); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("WriteToBytes: %v", err)
	}
	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("create outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_credit_note.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("save local copy: %v", err)
	}

	fileName := zip.DocumentFileName(zip.KindCreditNote, cn.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(cn.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)

	client := soap.New(soap.HabilitacionURL, cert, key)
	result, err := client.SendTestSetAsync(zipFileName, zipBytes, env["DIAN_TEST_SET_ID"])
	if err != nil {
		t.Fatalf("SendTestSetAsync: %v", err)
	}

	t.Logf("CUDE: %s", cn.CUFE)
	t.Logf("ZipKey: %q", result.ZipKey)
	if result.ErrorMessageList != nil {
		for _, e := range result.ErrorMessageList.Items {
			t.Logf("Initial error: file=%s message=%s success=%v", e.XmlFileName, e.ProcessedMessage, e.Success)
		}
	}
	if result.ZipKey == "" {
		t.Fatal("DIAN did not return a ZipKey — check ErrorMessageList above")
	}

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
	t.Log("no result from GetStatusZip within this test's wait time")
}
