package integrationtest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cuds"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
)

// TestSendTestSetAsync_SupportDocument_Real builds, signs, compresses, and sends a real
// Support Document to the DIAN certification environment, the same way TestSendTestSetAsync_Real
// does for the Electronic Sales Invoice. It uses the same credentials (same software, same PIN)
// but the Support Document test set (a different TestSetID) and reversed roles: Supplier = the
// third party not required to invoice, Customer = the issuing company.
//
// The goal is to isolate whether the DSAD06 issue lies in cofacture or in apidian.
func TestSendTestSetAsync_SupportDocument_Real(t *testing.T) {
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

	// Number unique per run (offset within the Support Document range) so the test can be
	// re-run without the DIAN rejecting it for a duplicate number.
	number := fmt.Sprintf("%d", 984000000+(time.Now().Unix()%1000000))

	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1: documento soporte en adquisiciones efectuadas a no obligados a facturar.",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10", // Resident
		DocumentTypeCode:  "05",
		HashType:          "CUDS-SHA384",

		Prefix: "SEDS",
		Number: number,

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

		// Roles reversed for the Support Document: Supplier = the third party NOT REQUIRED to
		// invoice (SNO), Customer = the issuing company (ABS).
		// The DIAN requires schemeName="31" (NIT) for the SNO — verified against DS-real.xml.
		Supplier: domain.Party{
			EntityTypeCode: "2",
			Identification: domain.Identification{
				Number:           "1020304050",
				TypeCode:         "31",
				VerificationCode: "8",
			},
			Name: "María García",
			Address: domain.Address{
				Line:        "Vereda El Rosal",
				CityCode:    "05001",
				CityName:    "Medellín",
				PostalZone:  "050001",
				StateCode:   "05",
				StateName:   "Antioquia",
				CountryCode: "CO",
				CountryName: "Colombia",
			},
			LiabilityCodes: []string{"R-99-PN"},
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
		},

		// Customer = the purchasing/issuing company for the Support Document (Diego).
		Customer: domain.Party{
			EntityTypeCode: "2",
			Identification: domain.Identification{
				Number:           "6382356",
				TypeCode:         "31",
				VerificationCode: "7",
			},
			Name: "MONTOYA VALLEJO DIEGO FERNANDO",
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
		},

		PaymentMeans: []domain.PaymentMean{{Code: "1", PaymentMethodCode: "10"}},

		HeaderTaxes: []domain.Tax{
			{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
		},

		Totals: domain.Totals{
			LineExtensionCents: 10000,
			TaxExclusiveCents:  10000,
			TaxInclusiveCents:  11900,
			PayableCents:       11900,
		},

		Lines: []domain.Line{{
			Description:        "Servicio de prueba DS (TestSendTestSetAsync_SupportDocument_Real)",
			Quantity:           1,
			UnitCode:           "94",
			LineExtensionCents: 10000,
			UnitPriceCents:     10000,
			ItemCode:           "0001",
			ItemTypeCode:       "999",
			ItemTypeName:       "Estándar de adopción del contribuyente",
			Taxes: []domain.Tax{
				{TaxableAmountCents: 10000, TaxAmountCents: 1900, Percent: 19, TypeCode: "01", TypeName: "IVA"},
			},
		}},

		NumberingRange: domain.NumberingRange{
			AuthorizedCode: "18760000001",
			Prefix:         "SEDS",
			StartNumber:    "984000000",
			EndNumber:      "985000000",
			StartDate:      "2026-01-01",
			EndDate:        "2026-12-31",
		},

		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "31", VerificationCode: "7"},
			SoftwareID:             env["DIAN_SOFTWARE_ID"],
		},
	}

	inv.CUFE = cuds.Compute(inv, env["DIAN_PIN"])
	t.Logf("CUDS: %s", inv.CUFE)
	inv.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], inv.Prefix+inv.Number)
	inv.QRURL = qr.SupportDocumentContent(inv, inv.CUFE, env["DIAN_PIN"])

	doc, err := builder.BuildSupportDocument(inv)
	if err != nil {
		t.Fatalf("BuildSupportDocument: %v", err)
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
	if err := os.WriteFile(filepath.Join(outputsDir, "_support_document.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("save local copy of the XML: %v", err)
	}

	// The issuer's NIT (Customer in a Support Document) is used for the ZIP file name — just
	// like the Electronic Sales Invoice uses the Supplier's NIT (which IS the issuer for that
	// document type).
	companyNIT := inv.Customer.Identification.Number
	fileName := zip.DocumentFileName(zip.KindSupportDocument, companyNIT, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(companyNIT, zip.SoftwarePropioCode, now.Year(), 1)

	client := soap.New(soap.HabilitacionURL, cert, key)

	// TestSetID specific to the Support Document — different from the invoice's.
	const dsTestSetID = "ffad5a13-987b-4cab-9f46-8994d6643602"

	result, err := client.SendTestSetAsync(zipFileName, zipBytes, dsTestSetID)
	if err != nil {
		t.Fatalf("SendTestSetAsync: %v", err)
	}

	t.Logf("ZipKey: %q", result.ZipKey)
	if result.ErrorMessageList != nil {
		for _, e := range result.ErrorMessageList.Items {
			t.Logf("Initial error: file=%s message=%s success=%v", e.XmlFileName, e.ProcessedMessage, e.Success)
		}
	}

	if result.ZipKey == "" {
		t.Fatal("DIAN did not return a ZipKey — check ErrorMessageList above")
	}

	testSetClosed := false
	for i := 0; i < 12; i++ {
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
		inProcess := false
		for _, st := range statuses {
			t.Logf("TestSetAsync attempt %d → IsValid=%v StatusCode=%q StatusDescription=%s",
				i+1, st.IsValid, st.StatusCode, st.StatusDescription)
			if st.ErrorMessage != nil {
				for _, m := range st.ErrorMessage.Items {
					t.Logf("  ErrorMessage: %s", m)
				}
			}
			if strings.Contains(st.StatusDescription, "Set de prueba") &&
				strings.Contains(st.StatusDescription, "se encuentra Aceptado") {
				testSetClosed = true
			}
			if strings.Contains(st.StatusDescription, "proceso de validación") {
				inProcess = true
			}
		}
		if !inProcess {
			break // final result received
		}
		t.Logf("still in progress, retrying in 5s…")
	}

	if !testSetClosed {
		t.Log("test set is not closed — DSAD06 not reproduced via SendBillSync")
		return
	}

	// The Support Document test set was already closed (0 documents required, auto-accepted).
	// Apidian detects this and calls SendBillSync with the same XML. We replicate that here
	// to see what the DIAN returns from the document's real validation.
	t.Log("test set closed → sending the same XML via SendBillSync (same as apidian)...")
	syncResult, err := client.SendBillSync(zipFileName, zipBytes)
	if err != nil {
		t.Fatalf("SendBillSync: %v", err)
	}
	t.Logf("SendBillSync → IsValid=%v StatusCode=%s StatusDescription=%s StatusMessage=%s XmlDocumentKey=%s",
		syncResult.IsValid, syncResult.StatusCode, syncResult.StatusDescription, syncResult.StatusMessage, syncResult.XmlDocumentKey)
	if syncResult.ErrorMessage != nil {
		for _, m := range syncResult.ErrorMessage.Items {
			t.Logf("  ErrorMessage: %s", m)
		}
	}
}
