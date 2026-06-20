package soap

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/diegofxm/ubl21dian/builder"
	"github.com/diegofxm/ubl21dian/cufe"
	"github.com/diegofxm/ubl21dian/domain"
	"github.com/diegofxm/ubl21dian/qr"
	"github.com/diegofxm/ubl21dian/securitycode"
	"github.com/diegofxm/ubl21dian/signer"
	"github.com/diegofxm/ubl21dian/zip"
)

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

// TestSendTestSetAsync_Real construye, firma, comprime y envía una factura real al
// ambiente de habilitación de la DIAN, usando las credenciales reales de
// docs/reference. Se omite por defecto (mismo patrón que core-bank con DATABASE_URL) —
// solo corre si UBL21DIAN_TEST_FIXTURES_DIR apunta a esa carpeta.
//
// Esto NO completa el proceso de habilitación por sí solo — eso requiere enviar el "set de
// pruebas" completo que la DIAN define para cada participante, no una factura inventada.
// El objetivo aquí es validar que el pipeline propio (build → CUFE → firma → zip → envío)
// es aceptado por el servidor real, no terminar la habilitación de una vez.
func TestSendTestSetAsync_Real(t *testing.T) {
	dir := os.Getenv("UBL21DIAN_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("UBL21DIAN_TEST_FIXTURES_DIR no configurado, se omite la prueba real contra la DIAN")
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

	rangeFrom, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_FROM"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_FROM: %v", err)
	}
	rangeTo, err := time.Parse("02-01-2006", env["DIAN_RANGE_DATE_TO"])
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_DATE_TO: %v", err)
	}

	now := time.Now().In(domain.Bogota)
	// El número debe caer dentro del rango autorizado (DIAN_RANGE_FROM..DIAN_RANGE_TO) — la
	// DIAN rechazó la primera vuelta de esta prueba con FAD05b al usar "1". Se suma un
	// desplazamiento pequeño basado en la hora para poder re-correr esta prueba varias
	// veces sin repetir el mismo número de factura.
	rangeFromInt, err := strconv.ParseInt(env["DIAN_RANGE_FROM"], 10, 64)
	if err != nil {
		t.Fatalf("parsear DIAN_RANGE_FROM: %v", err)
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

		// HeaderTaxes con IVA al 0% (en vez de omitir TaxTotal por completo) — la DIAN
		// rechazó la primera vuelta de esta prueba con "FAU04: Base Imponible es distinto
		// a la suma de los valores de las bases imponibles de todas líneas de detalle"
		// cuando no se informaba ningún TaxTotal.
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

		// La DIAN exige schemeName="31" (NIT) para el ProviderID del proveedor tecnológico
		// incluso cuando, como aquí, el emisor es persona natural con cédula en el resto
		// del documento (regla FAB23) — y el DV sí debe quedar correctamente calculado
		// (FAB22b), por eso "7" y no un valor cualquiera.
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

	// OJO: nunca llamar doc.Indent() (ni nada que reescriba el árbol) después de firmar — eso
	// inserta texto en blanco en la estructura y el documento que se transmite deja de ser
	// el mismo que se canonicalizó/firmó. Eso fue exactamente la causa de un "Valor de la
	// firma inválido" real de la DIAN en una vuelta anterior de esta prueba.
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("WriteToBytes: %v", err)
	}
	// Guardamos copia local del XML firmado que se envía, para poder inspeccionarlo aparte
	// si la DIAN lo rechaza (docs/reference/outputs está en .gitignore, no se commitea).
	outputsDir := filepath.Join(dir, "outputs")
	if err := os.MkdirAll(outputsDir, 0o755); err != nil {
		t.Fatalf("crear outputs/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputsDir, "_realsend_invoice.xml"), xmlBytes, 0o644); err != nil {
		t.Fatalf("guardar copia local del XML: %v", err)
	}

	fileName := zip.DocumentFileName(zip.KindInvoice, inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := zip.PackageFileName(inv.Supplier.Identification.Number, zip.SoftwarePropioCode, now.Year(), 1)

	client := New(HabilitacionURL, cert, key)
	result, err := client.SendTestSetAsync(zipFileName, zipBytes, env["DIAN_TEST_SET_ID"])
	if err != nil {
		t.Fatalf("SendTestSetAsync: %v", err)
	}

	t.Logf("CUFE: %s", inv.CUFE)
	t.Logf("ZipKey: %q", result.ZipKey)
	if result.ErrorMessageList != nil {
		for _, e := range result.ErrorMessageList.Items {
			t.Logf("Error inicial: archivo=%s mensaje=%s success=%v", e.XmlFileName, e.ProcessedMessage, e.Success)
		}
	}

	if result.ZipKey == "" {
		t.Fatal("la DIAN no devolvió ZipKey — revisar ErrorMessageList arriba")
	}

	// La validación es asíncrona; reintentamos GetStatusZip unos segundos antes de darnos
	// por vencidos a esperar el resultado dentro de esta misma prueba.
	for i := 0; i < 6; i++ {
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
		for _, st := range statuses {
			t.Logf("Resultado: IsValid=%v StatusCode=%s StatusMessage=%s StatusDescription=%s",
				st.IsValid, st.StatusCode, st.StatusMessage, st.StatusDescription)
			if st.ErrorMessage != nil {
				for _, m := range st.ErrorMessage.Items {
					t.Logf("  ErrorMessage: %s", m)
				}
			}
		}
		return
	}
	t.Log("no se obtuvo resultado de GetStatusZip dentro del tiempo de espera de esta prueba; el ZipKey queda registrado arriba para consultarlo después")
}
