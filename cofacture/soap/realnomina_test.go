package soap

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
	"github.com/diegofxm/cofacture/zip"
)

// TestSendNomina_Real construye, firma, comprime y envía una NominaIndividual real al
// ambiente de habilitación DIAN usando las credenciales de docs/reference.
// Solo corre si COFACTURE_TEST_FIXTURES_DIR apunta a esa carpeta.
func TestSendNomina_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR no configurado, se omite la prueba real contra DIAN")
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

	now := time.Now().In(bogota)

	// Consecutivo = solo la parte numérica. Numero = Prefijo + Consecutivo.
	consec := fmt.Sprintf("%d", now.Unix()%100000) // e.g. "87614"
	docNumber := "NE" + consec                     // e.g. "NE87614"

	// Datos del empleador (sujeto obligado) — misma persona natural que en los tests de FE.
	const empleadorNIT = "6382356"
	const empleadorDV = "7"

	// Datos del trabajador (empleado ficticio para prueba de habilitación).
	const trabajadorDoc = "1234567890"

	// Montos del periodo (enero 2024, 30 días).
	const (
		sueldo      = 1_423_500.0  // 1 SMLMV 2024 aprox
		diasTrab    = 30
		sueldoTrab  = sueldo // 30/30 días
		auxTransp   = 162_000.0 // auxilio de transporte 2024
		saludPct    = 4.0
		saludDed    = sueldoTrab * saludPct / 100
		pensionPct  = 4.0
		pensionDed  = sueldoTrab * pensionPct / 100
		devTotal    = sueldoTrab + auxTransp
		dedTotal    = saludDed + pensionDed
		compTotal   = devTotal - dedTotal
	)

	n := payroll.Nomina{
		Consecutivo: consec,    // solo la parte numérica: "87614"
		Prefijo:     "NE",
		Numero:      docNumber, // Prefijo + Consecutivo: "NE87614"
		TipoXML:     "102", // NominaIndividual
		Ambiente:    env["DIAN_ENVIRONMENT"],

		FechaGen:               now.Format("2006-01-02"),
		HoraGen:                now.Format("15:04:05-07:00"),
		FechaIngreso:           "2020-01-01",
		FechaLiquidacionInicio: "2024-01-01",
		FechaLiquidacionFin:    "2024-01-31",
		TiempoLaborado:         30,

		Pais:              "CO",
		DepartamentoEstado: "76",
		MunicipioCiudad:   "76520",
		Idioma:            "es",

		PeriodoNomina: 4, // Mensual
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

	// SoftwareSC: SHA-384(SoftwareID + PIN + NroDocumento) — sección 8.2 del Anexo.
	softwareSC := securitycode.Compute(n.SoftwareID, n.PIN, n.Numero)

	// CUNE: SHA-384(NumNE + FecNE + HorNE + ValDev + ValDed + ValTolNE + NitNE + DocEmp + TipoXML + PIN + Amb)
	// NitNE = Empleador/@NIT; DocEmp = Trabajador/@NumeroDocumento (sección 8.1)
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

	// CodigoQR: mismo endpoint que FE/DS.
	codigoQR := qr.URL(n.Ambiente, cune)

	t.Logf("CUNE: %s", cune)
	t.Logf("SoftwareSC: %s", softwareSC)
	t.Logf("CodigoQR: %s", codigoQR)

	// Construir árbol XML.
	doc, err := payroll.Build(n, cune, softwareSC, codigoQR)
	if err != nil {
		t.Fatalf("payroll.Build: %v", err)
	}

	// Localizar placeholder de firma e insertar la firma XAdES-EPES.
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

	// Guardar copia del XML firmado para inspección.
	outputsDir := filepath.Join(dir, "outputs")
	_ = os.MkdirAll(outputsDir, 0o755)
	outPath := filepath.Join(outputsDir, "_realsend_nomina.xml")
	_ = os.WriteFile(outPath, xmlBytes, 0o644)
	t.Logf("XML firmado guardado en: %s", outPath)

	// Empaquetar en ZIP.
	xmlFileName := payroll.XMLFileName(empleadorNIT, now.Year(), 1)
	zipBytes, err := zip.Build([]zip.File{{Name: xmlFileName, Content: xmlBytes}})
	if err != nil {
		t.Fatalf("zip.Build: %v", err)
	}
	zipFileName := payroll.ZIPFileName(empleadorNIT, now.Year(), 1)
	t.Logf("ZIP: %s / XML: %s", zipFileName, xmlFileName)

	// Enviar al servicio SendNominaSync de habilitación.
	client := New(HabilitacionURL, cert, key)
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
		t.Errorf("la DIAN rechazó el documento — ver errores arriba")
	}
}

// bogota es la zona horaria de Colombia.
var bogota = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.FixedZone("COT", -5*60*60)
	}
	return loc
}()
