package soap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diegofxm/cofacture/signer"
)

// realClient devuelve un Client apuntando al endpoint real de habilitación de la DIAN,
// cargando el certificado desde COFACTURE_TEST_FIXTURES_DIR. Llama t.Skip si la variable
// no está configurada; llama t.Fatal si el certificado no carga.
func realClient(t *testing.T) *Client {
	t.Helper()
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR no configurado, se omite la prueba real contra la DIAN")
	}
	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("leer certificado_cert.pem: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("leer certificado_key.pem: %v", err)
	}
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}
	return New(HabilitacionURL, cert, key)
}

// queryRanges llama GetNumberingRange en el endpoint dado e imprime los rangos devueltos.
func queryRanges(t *testing.T, label, baseURL, nit, softwareCode string) {
	t.Helper()
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	certPEM, _ := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	keyPEM, _ := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}
	c := New(baseURL, cert, key)

	got, err := c.GetNumberingRange(nit, nit, softwareCode)
	if err != nil {
		t.Logf("[%s] ERROR: %v", label, err)
		return
	}
	t.Logf("[%s] OperationCode        : %s", label, got.OperationCode)
	t.Logf("[%s] OperationDescription : %s", label, got.OperationDescription)
	t.Logf("[%s] Rangos encontrados   : %d", label, len(got.ResponseList))
	for i, r := range got.ResponseList {
		t.Logf("[%s] --- Rango %d ---", label, i+1)
		t.Logf("[%s]   ResolutionNumber : %s", label, r.ResolutionNumber)
		t.Logf("[%s]   ResolutionDate   : %s", label, r.ResolutionDate)
		t.Logf("[%s]   Prefix           : %s", label, r.Prefix)
		t.Logf("[%s]   FromNumber       : %d", label, r.FromNumber)
		t.Logf("[%s]   ToNumber         : %d", label, r.ToNumber)
		t.Logf("[%s]   ValidDateFrom    : %s", label, r.ValidDateFrom)
		t.Logf("[%s]   ValidDateTo      : %s", label, r.ValidDateTo)
		t.Logf("[%s]   TechnicalKey     : %s", label, r.TechnicalKey)
	}
}

// TestGetNumberingRange_Real consulta ambos ambientes (habilitación y producción) para el
// mismo NIT y software. Los entornos son bases de datos separadas en la DIAN: una resolución
// autorizada en producción no aparece en habilitación y viceversa. Si SETP (FE) solo aparece
// en producción y SEDS (DS) solo en habilitación, eso es el comportamiento correcto.
func TestGetNumberingRange_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR no configurado, se omite la prueba real contra la DIAN")
	}
	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	nit := "6382356"
	softwareCode := env["DIAN_SOFTWARE_ID"]
	if softwareCode == "" {
		t.Fatal("DIAN_SOFTWARE_ID no está en credenciales.txt")
	}

	t.Log("=== HABILITACIÓN ===")
	queryRanges(t, "HAB", HabilitacionURL, nit, softwareCode)

	t.Log("=== PRODUCCIÓN ===")
	queryRanges(t, "PRD", ProduccionURL, nit, softwareCode)
}

// TestGetAcquirer_Real llama al webservice real de la DIAN con la cédula del emisor de
// prueba (6382356, tipo "13") — confirmado en la sesión 9.41 que responde StatusCode "404"
// con Message "El adquirente No existe en la base de datos" (resultado normal: no tiene
// correo/nombre registrado para recibir documentos electrónicos). El test pasa si la llamada
// no devuelve un error de Go — el StatusCode del body es informativo, no un fallo.
func TestGetAcquirer_Real(t *testing.T) {
	c := realClient(t)

	got, err := c.GetAcquirer("13", "6382356")
	if err != nil {
		t.Fatalf("GetAcquirer: %v", err)
	}

	t.Logf("StatusCode    : %s", got.StatusCode)
	t.Logf("Message       : %s", got.Message)
	t.Logf("ReceiverName  : %q", got.ReceiverName)
	t.Logf("ReceiverEmail : %q", got.ReceiverEmail)
}
