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

// TestGetNumberingRange_Real llama al webservice real de la DIAN con las credenciales del
// emisor de prueba para obtener sus rangos de numeración autorizados. Requiere
// COFACTURE_TEST_FIXTURES_DIR y que credenciales.txt tenga DIAN_SOFTWARE_ID.
// El resultado imprime cada rango con su resolución, prefijo, from/to, fechas y TechnicalKey.
func TestGetNumberingRange_Real(t *testing.T) {
	c := realClient(t)
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	nit := "6382356"
	softwareCode := env["DIAN_SOFTWARE_ID"]
	if softwareCode == "" {
		t.Fatal("DIAN_SOFTWARE_ID no está en credenciales.txt")
	}

	got, err := c.GetNumberingRange(nit, nit, softwareCode)
	if err != nil {
		t.Fatalf("GetNumberingRange: %v", err)
	}

	t.Logf("OperationCode        : %s", got.OperationCode)
	t.Logf("OperationDescription : %s", got.OperationDescription)
	t.Logf("Rangos encontrados   : %d", len(got.ResponseList))
	for i, r := range got.ResponseList {
		t.Logf("--- Rango %d ---", i+1)
		t.Logf("  ResolutionNumber : %s", r.ResolutionNumber)
		t.Logf("  ResolutionDate   : %s", r.ResolutionDate)
		t.Logf("  Prefix           : %s", r.Prefix)
		t.Logf("  FromNumber       : %d", r.FromNumber)
		t.Logf("  ToNumber         : %d", r.ToNumber)
		t.Logf("  ValidDateFrom    : %s", r.ValidDateFrom)
		t.Logf("  ValidDateTo      : %s", r.ValidDateTo)
		t.Logf("  TechnicalKey     : %s", r.TechnicalKey)
	}
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
