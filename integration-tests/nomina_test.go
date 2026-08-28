package integrationtest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/payroll"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"
)

// TestSendNomina_Real builds, signs, compresses, and sends a real Individual Payroll document
// to the DIAN certification environment using the credentials from docs/reference.
// It only runs if COFACTURE_TEST_FIXTURES_DIR points to that folder.
func TestSendNomina_Real(t *testing.T) {
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

	now := time.Now().In(bogota)

	// Consecutivo = numeric part only. Numero = Prefijo + Consecutivo.
	consec := fmt.Sprintf("%d", now.Unix()%100000) // e.g. "87614"
	docNumber := "NE" + consec                     // e.g. "NE87614"

	// Employer data (the party responsible for the payroll) — same natural person used in the
	// Electronic Sales Invoice tests.
	const empleadorNIT = "6382356"
	const empleadorDV = "7"

	// Worker data (a fictitious employee for certification-environment testing).
	const trabajadorDoc = "1234567890"

	// Amounts for the period (January 2024, 30 days).
	const (
		sueldo     = 1_423_500.0 // approx. 1 SMLMV (minimum monthly wage) for 2024
		diasTrab   = 30
		sueldoTrab = sueldo    // 30/30 days worked
		auxTransp  = 162_000.0 // 2024 transportation allowance
		saludPct   = 4.0
		saludDed   = sueldoTrab * saludPct / 100
		pensionPct = 4.0
		pensionDed = sueldoTrab * pensionPct / 100
		devTotal   = sueldoTrab + auxTransp
		dedTotal   = saludDed + pensionDed
		compTotal  = devTotal - dedTotal
	)

	n := payroll.Nomina{
		Consecutivo: consec, // numeric part only: "87614"
		Prefijo:     "NE",
		Numero:      docNumber, // Prefijo + Consecutivo: "NE87614"
		TipoXML:     "102",     // Individual Payroll
		Ambiente:    env["DIAN_ENVIRONMENT"],

		FechaGen:               now.Format("2006-01-02"),
		HoraGen:                now.Format("15:04:05-07:00"),
		FechaIngreso:           "2020-01-01",
		FechaLiquidacionInicio: "2024-01-01",
		FechaLiquidacionFin:    "2024-01-31",
		TiempoLaborado:         30,

		Pais:               "CO",
		DepartamentoEstado: "76",
		MunicipioCiudad:    "76520",
		Idioma:             "es",

		PeriodoNomina: 4, // Monthly
		TipoMoneda:    "COP",
		TRM:           "1.00",

		SoftwareID: env["DIAN_SOFTWARE_ID"],
		PIN:        env["DIAN_PIN"],

		ProveedorNIT:         empleadorNIT,
		ProveedorDV:          empleadorDV,
		ProveedorRazonSocial: "DIEGO FERNANDO MONTOYA VALLEJO",
		ProveedorApellido1:   "MONTOYA",
		ProveedorApellido2:   "VALLEJO",
		ProveedorNombre1:     "DIEGO",
		ProveedorNombre2:     "FERNANDO",

		FechasPago: []string{"2024-01-31"},

		Empleador: payroll.Empleador{
			RazonSocial:     "DIEGO FERNANDO MONTOYA VALLEJO",
			PrimerApellido:  "MONTOYA",
			SegundoApellido: "VALLEJO",
			PrimerNombre:    "DIEGO",
			OtrosNombres:    "FERNANDO",
			NIT:             empleadorNIT,
			DV:              empleadorDV,
			Pais:            "CO",
			Departamento:    "76",
			Municipio:       "76520",
			Direccion:       "CL 13 A 25 26 BRR LAS AMERICAS",
		},

		Trabajador: payroll.Trabajador{
			TipoTrabajador:    "01",
			SubTipoTrabajador: "00",
			AltoRiesgoPension: false,
			TipoDocumento:     "13",
			NumeroDocumento:   trabajadorDoc,
			PrimerApellido:    "PEREZ",
			SegundoApellido:   "GARCIA",
			PrimerNombre:      "JUAN",
			OtrosNombres:      "CARLOS",
			LugarPais:         "CO",
			LugarDepartamento: "76",
			LugarMunicipio:    "76520",
			LugarDireccion:    "CL 1 2 3",
			SalarioIntegral:   false,
			TipoContrato:      "2",
			Sueldo:            sueldo,
			CodigoTrabajador:  "001",
		},

		Pago: payroll.Pago{
			Forma:        "1",
			Metodo:       "10",
			Banco:        "",
			TipoCuenta:   "",
			NumeroCuenta: "",
		},

		Devengados: payroll.Devengados{
			Basico: payroll.Basico{
				DiasTrabajados:  diasTrab,
				SueldoTrabajado: sueldoTrab,
			},
			Transporte: &payroll.Transporte{
				AuxilioTransporte: auxTransp,
				ViaticoManuAlojS:  0,
				ViaticoManuAlojNS: 0,
			},
		},

		Deducciones: payroll.Deducciones{
			Salud: &payroll.DeduccionPct{
				Porcentaje: saludPct,
				Deduccion:  saludDed,
			},
			FondoPension: &payroll.DeduccionPct{
				Porcentaje: pensionPct,
				Deduccion:  pensionDed,
			},
		},

		DevengadosTotal:  devTotal,
		DeduccionesTotal: dedTotal,
		ComprobanteTotal: compTotal,
	}

	// SoftwareSC: SHA-384(SoftwareID + PIN + NroDocumento) — Technical Annex section 8.2.
	softwareSC := securitycode.Compute(n.SoftwareID, n.PIN, n.Numero)

	// CUNE: SHA-384(NumNE + FecNE + HorNE + ValDev + ValDed + ValTolNE + NitNE + DocEmp + TipoXML + PIN + Amb)
	// NitNE = Empleador/@NIT; DocEmp = Trabajador/@NumeroDocumento (section 8.1)
	cune := payroll.Cune(
		n.Numero,
		n.FechaGen,
		n.HoraGen,
		fmt.Sprintf("%.2f", n.DevengadosTotal),
		fmt.Sprintf("%.2f", n.DeduccionesTotal),
		fmt.Sprintf("%.2f", n.ComprobanteTotal),
		n.Empleador.NIT,
		n.Trabajador.NumeroDocumento,
		n.TipoXML,
		n.PIN,
		n.Ambiente,
	)

	// CodigoQR: same endpoint used for the Electronic Sales Invoice / Support Document.
	codigoQR := qr.URL(n.Ambiente, cune)

	t.Logf("CUNE: %s", cune)
	t.Logf("SoftwareSC: %s", softwareSC)
	t.Logf("CodigoQR: %s", codigoQR)

	// Build the XML tree.
	doc, err := payroll.Build(n, cune, softwareSC, codigoQR)
	if err != nil {
		t.Fatalf("payroll.Build: %v", err)
	}

	// Locate the signature placeholder and insert the XAdES-EPES signature.
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		t.Fatalf("SignaturePlaceholder: %v", err)
	}
	if err := signer.New(cert, key).Sign(doc.Root(), placeholder, "supplier", now); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		t.Fatalf("WriteToBytes: %v", err)
	}

	// Save a copy of the signed XML for inspection.
	outputsDir := filepath.Join(dir, "outputs")
	_ = os.MkdirAll(outputsDir, 0o755)
	outPath := filepath.Join(outputsDir, "_nomina.xml")
	_ = os.WriteFile(outPath, xmlBytes, 0o644)
	t.Logf("signed XML saved to: %s", outPath)

	// Package into a ZIP.
	xmlFileName := payroll.XMLFileName(empleadorNIT, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: xmlFileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := payroll.ZIPFileName(empleadorNIT, now.Year(), 1)
	t.Logf("ZIP: %s / XML: %s", zipFileName, xmlFileName)

	// Send to the certification-environment SendNominaSync service.
	client := soap.New(soap.HabilitacionURL, cert, key)
	result, err := client.SendNominaSyncTestSet(zipBytes, env["PAYROLL_TEST_SET_ID"])
	if err != nil {
		t.Fatalf("SendNominaSyncTestSet: %v", err)
	}

	t.Logf("IsValid: %v", result.IsValid)
	t.Logf("StatusCode: %s", result.StatusCode)
	t.Logf("StatusMessage: %s", result.StatusMessage)
	t.Logf("StatusDescription: %s", result.StatusDescription)
	t.Logf("XmlDocumentKey: %s", result.XmlDocumentKey)

	if result.ErrorMessage != nil {
		for _, m := range result.ErrorMessage.Items {
			t.Logf("Error: %s", m)
		}
	}

	if !result.IsValid {
		t.Errorf("DIAN rejected the document — see errors above")
	}
}

// bogota is Colombia's time zone.
var bogota = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.FixedZone("COT", -5*60*60)
	}
	return loc
}()
