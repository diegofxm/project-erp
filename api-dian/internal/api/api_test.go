package api_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"software.sslmate.com/src/go-pkcs12"

	"github.com/diegofxm/api-dian/internal/api"
	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
)

// ── Setup ────────────────────────────────────────────────────────────────────────────────────

type testEnv struct {
	handler http.Handler
}

func selfSignedP12Base64(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "api-dian test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	p12, err := pkcs12.Modern.Encode(key, cert, nil, "clave-de-prueba")
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(p12)
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	log := zap.NewNop()

	issuerSvc := issuers.New(issuers.NewMemoryRepository())
	numberingSvc := numbering.New(numbering.NewMemoryRepository())
	docsSvc := documents.New(documents.NewMemoryRepository(), issuerSvc, numberingSvc)

	a := api.NewFromServices(log, issuerSvc, numberingSvc, docsSvc)

	return &testEnv{handler: a.Handler()}
}

func (e *testEnv) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&b).Encode(body))
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	e.handler.ServeHTTP(rw, req)
	return rw
}

func decode(t *testing.T, rw *httptest.ResponseRecorder, dst any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rw.Body).Decode(dst))
}

// ── Issuers ──────────────────────────────────────────────────────────────────────────────────

func TestAPI_CreateIssuer_OK(t *testing.T) {
	env := newTestEnv(t)

	rw := env.do(t, "POST", "/api/v1/issuers", map[string]any{
		"nit":                      "900373076",
		"check_digit":              "1",
		"business_name":            "Empresa de Prueba S.A.S.",
		"identification_type_code": "31",
		"department_code":          "11",
		"municipality_code":        "11001",
		"address_line":             "Calle 1 # 2-3",
		"email":                    "facturacion@empresa.test",
		"environment":              "2",
		"software_id":              "software-id-de-prueba",
		"software_pin":             "12345",
		"certificate_base64":       selfSignedP12Base64(t),
		"certificate_password":     "clave-de-prueba",
	})

	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "900373076", got["nit"])
	assert.NotContains(t, got, "certificate_base64", "el secreto nunca debe salir en la respuesta")
	assert.NotContains(t, got, "software_pin", "el secreto nunca debe salir en la respuesta")
}

func TestAPI_CreateIssuer_InvalidJSON(t *testing.T) {
	env := newTestEnv(t)
	req := httptest.NewRequest("POST", "/api/v1/issuers", bytes.NewBufferString("{not json"))
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_GetIssuer_NotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/issuers/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetIssuer_InvalidUUID(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/issuers/no-es-un-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_CreateIssuer_MissingRequiredField(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/issuers", map[string]any{
		"business_name": "Sin NIT",
	})
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

// ── Numbering ranges ─────────────────────────────────────────────────────────────────────────

func createTestIssuer(t *testing.T, env *testEnv) string {
	t.Helper()
	rw := env.do(t, "POST", "/api/v1/issuers", map[string]any{
		"nit":                      "900373076",
		"check_digit":              "1",
		"business_name":            "Empresa de Prueba S.A.S.",
		"identification_type_code": "31",
		"department_code":          "11",
		"municipality_code":        "11001",
		"address_line":             "Calle 1 # 2-3",
		"email":                    "facturacion@empresa.test",
		// Producción a propósito: desde que existe SendBillSync, habilitación SIEMPRE
		// intenta enviar de verdad (con o sin TestSetID) — estas son pruebas con httptest,
		// sin red real. Ver el mismo comentario en internal/documents/service_test.go.
		"environment":          "1",
		"software_id":          "software-id-de-prueba",
		"software_pin":         "12345",
		"certificate_base64":   selfSignedP12Base64(t),
		"certificate_password": "clave-de-prueba",
	})
	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	return got["id"].(string)
}

func TestAPI_CreateNumberingRange_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/issuers/"+issuerID+"/numbering-ranges", map[string]any{
		"dian_document_type_code": "01",
		"prefix":                  "SETP",
		"resolution_number":       "18760000001",
		"resolution_date":         "2024-01-01",
		"range_from":              1,
		"range_to":                5000,
		"valid_from":              "2024-01-01",
		"valid_to":                "2026-01-01",
		"environment":             "2",
		"technical_key":           "clave-tecnica-de-prueba",
	})

	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "SETP", got["prefix"])
	assert.NotContains(t, got, "technical_key", "el secreto nunca debe salir en la respuesta")
}

func TestAPI_CreateNumberingRange_InvalidDate(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/issuers/"+issuerID+"/numbering-ranges", map[string]any{
		"dian_document_type_code": "01",
		"prefix":                  "SETP",
		"resolution_number":       "18760000001",
		"resolution_date":         "no-es-una-fecha",
		"range_from":              1,
		"valid_from":              "2024-01-01",
		"valid_to":                "2026-01-01",
		"environment":             "2",
	})
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_GetNumberingRange_NotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/numbering-ranges/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

// ── Documents ────────────────────────────────────────────────────────────────────────────────

func createTestRange(t *testing.T, env *testEnv, issuerID, docType, prefix string) string {
	t.Helper()
	rw := env.do(t, "POST", "/api/v1/issuers/"+issuerID+"/numbering-ranges", map[string]any{
		"dian_document_type_code": docType,
		"prefix":                  prefix,
		"resolution_number":       "18760000001",
		"resolution_date":         "2024-01-01",
		"range_from":              1,
		"range_to":                5000,
		"valid_from":              "2024-01-01",
		"valid_to":                "2026-01-01",
		"environment":             "1", // mismo criterio que createTestIssuer — sin red real
		"technical_key":           "clave-tecnica-de-prueba",
	})
	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	return got["id"].(string)
}

func testCustomer() map[string]any {
	return map[string]any{
		"identification": map[string]any{"number": "222222222222", "type_code": "13"},
		"name":           "Consumidor Final",
	}
}

func testLines() []map[string]any {
	return []map[string]any{{
		"description":          "Servicio de prueba",
		"quantity":             1,
		"unit_code":            "94",
		"line_extension_cents": 10000,
		"unit_price_cents":     10000,
		"taxes": []map[string]any{
			{"taxable_amount_cents": 10000, "tax_amount_cents": 0, "percent": 0, "type_code": "01", "type_name": "IVA"},
		},
	}}
}

func TestAPI_IssueInvoice_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)
	rangeID := createTestRange(t, env, issuerID, "01", "SETP")

	rw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"issuer_id":          issuerID,
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})

	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "SETP", got["prefix"])
	assert.Equal(t, "built", got["status"])
	assert.NotEmpty(t, got["document_key"])
	assert.Contains(t, got["signed_xml"], "<ds:Signature")
}

func TestAPI_IssueInvoice_WrongDocumentType(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)
	rangeID := createTestRange(t, env, issuerID, "91", "SETPNC") // rango de Nota Crédito

	rw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"issuer_id":          issuerID,
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rw.Code)
}

func TestAPI_IssueCreditNote_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)
	invoiceRangeID := createTestRange(t, env, issuerID, "01", "SETP")
	cnRangeID := createTestRange(t, env, issuerID, "91", "SETPNC")

	invRw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"issuer_id":          issuerID,
		"numbering_range_id": invoiceRangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, invRw.Code)
	var inv map[string]any
	decode(t, invRw, &inv)

	rw := env.do(t, "POST", "/api/v1/credit-notes", map[string]any{
		"issuer_id":             issuerID,
		"numbering_range_id":    cnRangeID,
		"customer":              testCustomer(),
		"lines":                 testLines(),
		"credit_note_type_code": "2",
		"billing_reference": map[string]any{
			"prefix":     inv["prefix"],
			"number":     "1",
			"cufe":       inv["document_key"],
			"issue_date": "2026-06-20",
		},
	})

	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "SETPNC", got["prefix"])
	assert.Equal(t, "91", got["dian_document_type_code"])
}

func TestAPI_IssueCreditNote_MissingBillingReference(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)
	cnRangeID := createTestRange(t, env, issuerID, "91", "SETPNC")

	rw := env.do(t, "POST", "/api/v1/credit-notes", map[string]any{
		"issuer_id":             issuerID,
		"numbering_range_id":    cnRangeID,
		"customer":              testCustomer(),
		"lines":                 testLines(),
		"credit_note_type_code": "2",
	})

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_GetDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/documents/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetDocument_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID := createTestIssuer(t, env)
	rangeID := createTestRange(t, env, issuerID, "01", "SETP")

	createRw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"issuer_id":          issuerID,
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.do(t, "GET", "/api/v1/documents/"+created["id"].(string), nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, created["document_key"], got["document_key"])
}

// ── Middleware ───────────────────────────────────────────────────────────────────────────────

func TestAPI_RequestIDHeader(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/issuers/"+uuid.New().String(), nil)
	assert.NotEmpty(t, rw.Header().Get("X-Request-ID"))
}

func TestAPI_UnknownRoute_404(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/no-existe", nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}
