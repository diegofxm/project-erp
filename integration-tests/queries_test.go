package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
)

// realClient returns a Client pointing at the DIAN's real certification-environment endpoint,
// loading the certificate from COFACTURE_TEST_FIXTURES_DIR. Calls t.Skip if the variable
// isn't set; calls t.Fatal if the certificate fails to load.
func realClient(t *testing.T) *soap.Client {
	t.Helper()
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping real-DIAN test")
	}
	certPEM, err := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	if err != nil {
		t.Fatalf("read certificado_cert.pem: %v", err)
	}
	keyPEM, err := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	if err != nil {
		t.Fatalf("read certificado_key.pem: %v", err)
	}
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}
	return soap.New(soap.HabilitacionURL, cert, key)
}

// queryRanges calls GetNumberingRange against the given endpoint and logs the returned ranges.
func queryRanges(t *testing.T, label, baseURL, nit, softwareCode string) {
	t.Helper()
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	certPEM, _ := os.ReadFile(filepath.Join(dir, "certificado_cert.pem"))
	keyPEM, _ := os.ReadFile(filepath.Join(dir, "certificado_key.pem"))
	cert, key, err := signer.LoadPEM(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("LoadPEM: %v", err)
	}
	c := soap.New(baseURL, cert, key)

	got, err := c.GetNumberingRange(nit, nit, softwareCode)
	if err != nil {
		t.Logf("[%s] ERROR: %v", label, err)
		return
	}
	t.Logf("[%s] OperationCode        : %s", label, got.OperationCode)
	t.Logf("[%s] OperationDescription : %s", label, got.OperationDescription)
	t.Logf("[%s] Ranges found         : %d", label, len(got.ResponseList))
	for i, r := range got.ResponseList {
		t.Logf("[%s] --- Range %d ---", label, i+1)
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

// TestGetNumberingRange_Real queries both environments (certification and production) for the
// same NIT and software. The environments are separate databases on the DIAN side: a resolution
// authorized in production does not appear in certification, and vice versa. If SETP (Electronic
// Sales Invoice) only appears in production and SEDS (Support Document) only in certification,
// that is the expected behavior.
func TestGetNumberingRange_Real(t *testing.T) {
	dir := os.Getenv("COFACTURE_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("COFACTURE_TEST_FIXTURES_DIR not set, skipping real-DIAN test")
	}
	env := parseEnvFile(t, filepath.Join(dir, "credenciales.txt"))

	nit := "6382356"
	softwareCode := env["DIAN_SOFTWARE_ID"]
	if softwareCode == "" {
		t.Fatal("DIAN_SOFTWARE_ID missing from credenciales.txt")
	}

	t.Log("=== CERTIFICATION ===")
	queryRanges(t, "HAB", soap.HabilitacionURL, nit, softwareCode)

	t.Log("=== PRODUCTION ===")
	queryRanges(t, "PRD", soap.ProduccionURL, nit, softwareCode)
}

// TestGetAcquirer_Real calls the DIAN's real web service with the test issuer's ID number
// (6382356, type "13") — confirmed to respond with StatusCode "404" and Message "El adquirente
// No existe en la base de datos" ("The acquirer does not exist in the database") — a normal
// result: no email/name registered to receive electronic documents. The test passes as long as
// the call doesn't return a Go error — the StatusCode in the body is informational, not a
// failure.
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
