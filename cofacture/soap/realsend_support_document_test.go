package soap

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
	"github.com/diegofxm/cofacture/zip"
)

// TestSendTestSetAsync_SupportDocument_Real construye, firma, comprime y envía un Documento
// Soporte real al ambiente de habilitación de la DIAN, igual que TestSendTestSetAsync_Real
// lo hace para FE. Usa las mismas credenciales (mismo software, mismo PIN) pero el set de
// pruebas DS (TestSetID distinto) y roles invertidos: Supplier = proveedor no obligado,
// Customer = empresa emisora.
//
// El objetivo es aislar si el problema de DSAD06 está en cofacture o en apidian.
func TestSendTestSetAsync_SupportDocument_Real(t *testing.T) {
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

	// Número único por ejecución (offset dentro del rango DS) para poder re-correr
	// el test sin que la DIAN rechace por número duplicado.
	number := fmt.Sprintf("%d", 984000000+(time.Now().Unix()%1000000))

	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1: documento soporte en adquisiciones efectuadas a no obligados a facturar.",
		EnvironmentCode:   env["DIAN_ENVIRONMENT"],
		OperationTypeCode: "10", // Residente
		DocumentTypeCode:  "05",
		HashType:          "CUDS-SHA384",

		Prefix: "SEDS",
		Number: number,

		IssueDate: now.Format("2006-01-02"),
		IssueTime: now.Format("15:04:05-07:00"),

		CurrencyCode: "COP",

		// Roles invertidos en DS: Supplier = proveedor NO OBLIGADO (SNO), Customer = empresa emisora (ABS).
		// La DIAN exige schemeName="31" (NIT) para el SNO — verificado en DS-real.xml.
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

		// Customer = empresa compradora/emisora del DS (Diego).
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
		t.Fatalf("crear outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_realsend_support_document.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("guardar copia local del XML: %v", err)
	}

	// El NIT del emisor (Customer en DS) se usa para el nombre del ZIP — igual que FE usa
	// el NIT del Supplier (que en FE ES el emisor).
	companyNIT := inv.Customer.Identification.Number
	fileName := zip.DocumentFileName(zip.KindSupportDocument, companyNIT, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(companyNIT, zip.SoftwarePropioCode, now.Year(), 1)

	client := New(HabilitacionURL, cert, key)

	// TestSetID específico para DS — distinto del de FE.
	const dsTestSetID = "ffad5a13-987b-4cab-9f46-8994d6643602"

	result, err := client.SendTestSetAsync(zipFileName, zipBytes, dsTestSetID)
	if err != nil {
		t.Fatalf("SendTestSetAsync: %v", err)
	}

	t.Logf("ZipKey: %q", result.ZipKey)
	if result.ErrorMessageList != nil {
		for _, e := range result.ErrorMessageList.Items {
			t.Logf("Error inicial: archivo=%s mensaje=%s success=%v", e.XmlFileName, e.ProcessedMessage, e.Success)
		}
	}

	if result.ZipKey == "" {
		t.Fatal("la DIAN no devolvió ZipKey — revisar ErrorMessageList arriba")
	}

	testSetClosed := false
	for i := 0; i < 12; i++ {
		time.Sleep(5 * time.Second)
		statuses, err := client.GetStatusZip(result.ZipKey)
		if err != nil {
			t.Logf("GetStatusZip intento %d: %v", i+1, err)
			continue
		}
		if len(statuses) == 0 {
			t.Logf("GetStatusZip intento %d: aún sin resultado", i+1)
			continue
		}
		inProcess := false
		for _, st := range statuses {
			t.Logf("TestSetAsync intento %d → IsValid=%v StatusCode=%q StatusDescription=%s",
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
			break // resultado definitivo recibido
		}
		t.Logf("aún en proceso, reintentando en 5s…")
	}

	if !testSetClosed {
		t.Log("el set de pruebas no está cerrado — DSAD06 no reproduced via SendBillSync")
		return
	}

	// El set DS ya estaba cerrado (0 documentos requeridos, aceptado automáticamente).
	// Apidian detecta esto y llama SendBillSync con el mismo XML. Lo replicamos aquí
	// para ver qué devuelve la DIAN en la validación real del documento.
	t.Log("set de pruebas cerrado → enviando el mismo XML via SendBillSync (igual que apidian)...")
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
