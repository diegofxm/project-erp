package soap

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/zip"
)

// TestSendBillSync_Real verifica si la DIAN sigue aceptando envíos contra habilitación a
// través de SendBillSync (envío normal, no atado a un TestSetID) ahora que el set de
// pruebas oficial de la Fase 1.7/1.9 ya quedó "Aceptado" — esa es justamente la pregunta que
// esta prueba responde: ¿el ambiente de habilitación sigue disponible para pruebas continuas
// usando las operaciones de envío normales, o la DIAN bloquea todo envío adicional una vez
// completada la certificación? Ver docs/api-dian-architecture.md sección 9.9 para el
// resultado y su interpretación.
func TestSendBillSync_Real(t *testing.T) {
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

	rangeFromDate, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_FROM: %v", err)
	}
	rangeToDate, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_TO: %v", err)
	}

	now := time.Now().In(domain.Bogota)
	rangeFromInt, err := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_FROM: %v", err)
	}
	// Mismo truco que TestSendTestSetAsync_Real: desplazamiento por hora para no repetir
	// número de factura entre corridas de esta prueba.
	number := strconv.FormatInt(rangeFromInt+2+time.Now().Unix()%100000, 10)

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
			Description:        "Servicio de prueba (TestSendBillSync_Real)",
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
			StartDate:      rangeFromDate.Format("2006-01-02"),
			EndDate:        rangeToDate.Format("2006-01-02"),
		},

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

	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("WriteToBytes: %v", err)
	}

	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("crear outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_realsend_sync_invoice.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("guardar copia local del XML: %v", err)
	}

	fileName := zip.DocumentFileName(zip.KindInvoice, inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)

	client := New(HabilitacionURL, cert, key)
	result, err := client.SendBillSync(zipFileName, zipBytes)
	if err != nil {
		t.Fatalf("SendBillSync: %v", err)
	}

	t.Logf("Prefix+Number: %s%s", inv.Prefix, inv.Number)
	t.Logf("CUFE: %s", inv.CUFE)
	t.Logf("IsValid: %v", result.IsValid)
	t.Logf("StatusCode: %s", result.StatusCode)
	t.Logf("StatusDescription: %s", result.StatusDescription)
	t.Logf("StatusMessage: %s", result.StatusMessage)
	t.Logf("XmlDocumentKey: %s", result.XmlDocumentKey)
	if result.ErrorMessage != nil {
		t.Logf("ErrorMessage: %v", result.ErrorMessage.Items)
	}
}
