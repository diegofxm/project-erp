package soap

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cuds"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/zip"
)

// TestSendBillSync_AdjustmentNote_Real construye, firma, comprime y envía una Nota de Ajuste
// al DS (NA, InvoiceTypeCode "95") vía SendBillSync (habilitación libre, sin test_set_id).
// Referencia el DS SEDS984000000 ya autorizado por la DIAN en habilitación.
//
// El objetivo es validar el builder (CreditNote root) y el flujo de envío antes de integrarlo
// en apidian, donde la DIAN rechazaba con "MessagesType not found" porque el builder generaba
// un <Invoice> en lugar de <CreditNote>.
func TestSendBillSync_AdjustmentNote_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR no configurado, se omite la prueba real contra la DIAN")
	}

	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("leer certificado: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("leer llave privada: %v", err)
	}
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}

	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	now := time.Now().In(domain.Bogota)

	// Número único por ejecución dentro del rango NAP (1–1000000).
	number := fmt.Sprintf("%d", 1+(time.Now().Unix()%999999))

	base := domain.Invoice{
		ProfileID:         "DIAN 2.1: Nota de ajuste al documento soporte en adquisiciones efectuadas a sujetos no obligados a expedir factura o documento equivalente",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10", // Residente (igual que el DS referenciado)
		DocumentTypeCode:  "95",
		HashType:          "CUDS-SHA384",

		Prefix: "NAP",
		Number: number,

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

		// Roles invertidos igual que DS: Supplier = SNO (proveedor no obligado), Customer = ABS (empresa).
		Supplier: domain.Party{
			EntityTypeCode: "2",
			Identification: domain.Identification{
				Number:           "1234567895",
				TypeCode:         "31",
				VerificationCode: "9",
			},
			Name: "Proveedor Prueba",
			Address: domain.Address{
				Line:        "CL 1 2 3",
				CityCode:    "86757",
				CityName:    "San Miguel",
				PostalZone:  "000000",
				StateCode:   "86",
				StateName:   "Putumayo",
				CountryCode: "CO",
				CountryName: "Colombia",
			},
			LiabilityCodes: []string{"R-99-PN"},
			TaxSchemeCode:  "ZZ",
			TaxSchemeName:  "No aplica",
		},

		// Customer = empresa emisora del DS (ABS = Diego).
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
				PostalZone:  "000000",
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

		// Valores simples con math exacto (19% * 10000 = 1900).
		// VLR02 exige NA ≤ DS ref (DS = 4225.07 COP); 119 COP está muy por debajo.
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
			Description:        "Ajuste al DS SEDS984000000 (TestSendBillSync_AdjustmentNote_Real)",
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

		SoftwareProvider: domain.SoftwareProvider{
			ProviderIdentification: domain.Identification{Number: "6382356", TypeCode: "31", VerificationCode: "7"},
			SoftwareID:             env["DIAN_SOFTWARE_ID"],
		},
	}

	an := domain.AdjustmentNote{
		Invoice: base,
		BillingReference: domain.BillingReference{
			Prefix:    "SEDS",
			Number:    "984000000",
			CUFE:      "aff9f22a7a3e419887d5065d69603dfc6ed7a20aa27d85f02a23b4c777194c85b4e947813e4e1149b5e90f347e835d64",
			IssueDate: "2026-07-19",
		},
		DiscrepancyResponse: &domain.DiscrepancyResponse{
			ReferenceID:  "SEDS984000000",
			ResponseCode: "2",
			Description:  "Ajuste al documento soporte",
		},
	}

	an.CUFE = cuds.Compute(an.Invoice, env["DIAN_PIN"])
	an.SoftwareSecurityCode = securitycode.Compute(env["DIAN_SOFTWARE_ID"], env["DIAN_PIN"], an.Prefix+an.Number)
	an.QRURL = qr.AdjustmentNoteContent(an.Invoice, an.CUFE, env["DIAN_PIN"])

	t.Logf("CUDS NA: %s", an.CUFE)

	doc, err := builder.BuildAdjustmentNote(an)
	if err != nil {
		t.Fatalf("BuildAdjustmentNote: %v", err)
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
		t.Fatalf("crear outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_realsend_adjustment_note.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("guardar copia local del XML: %v", err)
	}

	// NA usa el NIT del ABS (Customer) para el nombre del ZIP, igual que DS.
	companyNIT := an.Customer.Identification.Number
	fileName := zip.DocumentFileName(zip.KindAdjustmentNote, companyNIT, zip.SoftwarePropioCode, now.Year(), uint32(time.Now().Unix()%0xFFFFFFFF))
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(companyNIT, zip.SoftwarePropioCode, now.Year(), uint32(time.Now().Unix()%0xFFFFFFFF))

	client := New(HabilitacionURL, cert, key)

	// Habilitación libre — SendBillSync sin test_set_id (igual que NC/ND confirmados).
	resp, err := client.SendBillSync(zipFileName, zipBytes)
	if err != nil {
		t.Fatalf("SendBillSync: %v", err)
	}

	t.Logf("IsValid=%v StatusCode=%s StatusDescription=%s StatusMessage=%s XmlDocumentKey=%s",
		resp.IsValid, resp.StatusCode, resp.StatusDescription, resp.StatusMessage, resp.XmlDocumentKey)
	if resp.ErrorMessage != nil {
		for _, m := range resp.ErrorMessage.Items {
			t.Logf("  ErrorMessage: %s", m)
		}
	}

	if !resp.IsValid {
		t.Fatalf("la DIAN rechazó la NA: %s — %s", resp.StatusCode, resp.StatusDescription)
	}
}
