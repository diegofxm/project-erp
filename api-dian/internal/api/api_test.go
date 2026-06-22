package api_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	"github.com/diegofxm/api-dian/internal/auth"
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
	tokens := auth.NewTokenIssuer([]byte("clave-de-prueba-no-usar-en-produccion"))
	authSvc := auth.New(auth.NewMemoryRepository(), issuerSvc, tokens)

	a := api.NewFromServices(log, issuerSvc, numberingSvc, docsSvc, authSvc, tokens)

	return &testEnv{handler: a.Handler()}
}

func (e *testEnv) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return e.doAuth(t, method, path, "", body)
}

func (e *testEnv) doAuth(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&b).Encode(body))
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rw := httptest.NewRecorder()
	e.handler.ServeHTTP(rw, req)
	return rw
}

func decode(t *testing.T, rw *httptest.ResponseRecorder, dst any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(rw.Body).Decode(dst))
}

// ── Auth ─────────────────────────────────────────────────────────────────────────────────────

// registerRequest construye un payload de POST /api/v1/auth/register completo. email permite
// variar el correo entre llamadas — el mismo correo no puede registrarse dos veces. El NIT
// también se genera único por llamada (no solo el correo): cada registro crea un emisor
// nuevo, y dos emisores no pueden compartir NIT.
func registerRequest(t *testing.T, email string) map[string]any {
	nit := fmt.Sprintf("9%011d", time.Now().UnixNano()%100000000000)
	return map[string]any{
		"issuer": map[string]any{
			"nit":                      nit,
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
		},
		"email":    email,
		"password": "contraseña-segura",
		"name":     "Admin de Prueba",
	}
}

// registerTestIssuer registra un emisor + usuario nuevo y devuelve (issuerID, token).
func registerTestIssuer(t *testing.T, env *testEnv) (string, string) {
	t.Helper()
	email := fmt.Sprintf("admin-%s@empresa.test", uuid.New().String())
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest(t, email))
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())

	var got map[string]any
	decode(t, rw, &got)
	issuerID := got["issuer"].(map[string]any)["id"].(string)
	token := got["token"].(string)
	return issuerID, token
}

func TestAPI_Register_OK(t *testing.T) {
	env := newTestEnv(t)

	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest(t, "admin@empresa.test"))

	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotEmpty(t, got["token"])
	assert.Equal(t, "admin@empresa.test", got["user"].(map[string]any)["email"])
	assert.NotEmpty(t, got["issuer"].(map[string]any)["nit"])
	assert.NotContains(t, got["issuer"], "certificate_base64", "el secreto nunca debe salir en la respuesta")
	assert.NotContains(t, got["issuer"], "software_pin", "el secreto nunca debe salir en la respuesta")
}

func TestAPI_Register_InvalidJSON(t *testing.T) {
	env := newTestEnv(t)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString("{not json"))
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_Register_MissingRequiredField(t *testing.T) {
	env := newTestEnv(t)
	req := registerRequest(t, "admin@empresa.test")
	req["issuer"].(map[string]any)["nit"] = ""
	rw := env.do(t, "POST", "/api/v1/auth/register", req)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_Register_DuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	req := registerRequest(t, "admin@empresa.test")

	rw := env.do(t, "POST", "/api/v1/auth/register", req)
	require.Equal(t, http.StatusCreated, rw.Code)

	req2 := registerRequest(t, "admin@empresa.test")
	req2["issuer"].(map[string]any)["nit"] = "900111222" // distinto NIT, mismo correo
	rw = env.do(t, "POST", "/api/v1/auth/register", req2)
	assert.Equal(t, http.StatusConflict, rw.Code)
}

func TestAPI_Login_OK(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest(t, "admin@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)

	rw = env.do(t, "POST", "/api/v1/auth/login", map[string]any{
		"email":    "admin@empresa.test",
		"password": "contraseña-segura",
	})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotEmpty(t, got["token"])
}

func TestAPI_Login_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest(t, "admin@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)

	rw = env.do(t, "POST", "/api/v1/auth/login", map[string]any{
		"email":    "admin@empresa.test",
		"password": "contraseña-equivocada",
	})
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

// ── Acceso protegido ─────────────────────────────────────────────────────────────────────────

func TestAPI_GetMyIssuer_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "GET", "/api/v1/issuers/me", token, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, issuerID, got["id"])
}

func TestAPI_GetMyIssuer_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/issuers/me", nil)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_GetMyIssuer_InvalidToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.doAuth(t, "GET", "/api/v1/issuers/me", "esto-no-es-un-token", nil)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

// ── Numbering ranges ─────────────────────────────────────────────────────────────────────────

func createTestRange(t *testing.T, env *testEnv, token, docType, prefix string) string {
	t.Helper()
	rw := env.doAuth(t, "POST", "/api/v1/numbering-ranges", token, map[string]any{
		"dian_document_type_code": docType,
		"prefix":                  prefix,
		"resolution_number":       "18760000001",
		"resolution_date":         "2024-01-01",
		"range_from":              1,
		"range_to":                5000,
		"valid_from":              "2024-01-01",
		"valid_to":                "2026-01-01",
		"environment":             "1", // mismo criterio que registerRequest — sin red real
		"technical_key":           "clave-tecnica-de-prueba",
	})
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	return got["id"].(string)
}

func TestAPI_CreateNumberingRange_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "POST", "/api/v1/numbering-ranges", token, map[string]any{
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

func TestAPI_CreateNumberingRange_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/numbering-ranges", map[string]any{
		"dian_document_type_code": "01",
		"prefix":                  "SETP",
	})
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_CreateNumberingRange_InvalidDate(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "POST", "/api/v1/numbering-ranges", token, map[string]any{
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
	_, token := registerTestIssuer(t, env)
	rw := env.doAuth(t, "GET", "/api/v1/numbering-ranges/"+uuid.New().String(), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetNumberingRange_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	// El emisor B no debe poder ver el rango del emisor A — mismo 404 que si no existiera.
	rw := env.doAuth(t, "GET", "/api/v1/numbering-ranges/"+rangeID, tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_ListNumberingRanges_OnlyOwnAndFiltered(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	createTestRange(t, env, tokenA, "01", "SETP")
	createTestRange(t, env, tokenA, "91", "SETPNC")
	createTestRange(t, env, tokenB, "01", "SETP")

	rw := env.doAuth(t, "GET", "/api/v1/numbering-ranges", tokenA, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Len(t, got["numbering_ranges"].([]any), 2, "solo los rangos del emisor A")

	rw = env.doAuth(t, "GET", "/api/v1/numbering-ranges?dian_document_type_code=01", tokenA, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	decode(t, rw, &got)
	assert.Len(t, got["numbering_ranges"].([]any), 1)
}

func TestAPI_ListNumberingRanges_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/numbering-ranges", nil)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

// ── Documents ────────────────────────────────────────────────────────────────────────────────

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
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	rw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
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

func TestAPI_IssueInvoice_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"numbering_range_id": uuid.New().String(),
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_IssueInvoice_WrongDocumentType(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "91", "SETPNC") // rango de Nota Crédito

	rw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rw.Code)
}

func TestAPI_IssueInvoice_OtherTenantRange(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	// El emisor B no debe poder emitir usando el rango del emisor A.
	rw := env.doAuth(t, "POST", "/api/v1/invoices", tokenB, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rw.Code)
}

func TestAPI_IssueCreditNote_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	invoiceRangeID := createTestRange(t, env, token, "01", "SETP")
	cnRangeID := createTestRange(t, env, token, "91", "SETPNC")

	invRw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": invoiceRangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, invRw.Code)
	var inv map[string]any
	decode(t, invRw, &inv)

	rw := env.doAuth(t, "POST", "/api/v1/credit-notes", token, map[string]any{
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
	_, token := registerTestIssuer(t, env)
	cnRangeID := createTestRange(t, env, token, "91", "SETPNC")

	rw := env.doAuth(t, "POST", "/api/v1/credit-notes", token, map[string]any{
		"numbering_range_id":    cnRangeID,
		"customer":              testCustomer(),
		"lines":                 testLines(),
		"credit_note_type_code": "2",
	})

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_GetDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rw := env.doAuth(t, "GET", "/api/v1/documents/"+uuid.New().String(), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetDocument_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	createRw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "GET", "/api/v1/documents/"+created["id"].(string), token, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, created["document_key"], got["document_key"])
}

func TestAPI_GetDocument_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	createRw := env.doAuth(t, "POST", "/api/v1/invoices", tokenA, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	// El emisor B no debe poder ver el documento del emisor A — mismo 404 que si no existiera.
	rw := env.doAuth(t, "GET", "/api/v1/documents/"+created["id"].(string), tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_ListDocuments_OnlyOwnAndFiltered(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeA := createTestRange(t, env, tokenA, "01", "SETP")
	rangeB := createTestRange(t, env, tokenB, "01", "SETP")

	for i := 0; i < 2; i++ {
		rw := env.doAuth(t, "POST", "/api/v1/invoices", tokenA, map[string]any{
			"numbering_range_id": rangeA,
			"customer":           testCustomer(),
			"lines":              testLines(),
		})
		require.Equal(t, http.StatusCreated, rw.Code)
	}
	rw := env.doAuth(t, "POST", "/api/v1/invoices", tokenB, map[string]any{
		"numbering_range_id": rangeB,
		"customer":           testCustomer(),
		"lines":              testLines(),
	})
	require.Equal(t, http.StatusCreated, rw.Code)

	rw = env.doAuth(t, "GET", "/api/v1/documents", tokenA, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Len(t, got["documents"].([]any), 2, "solo los documentos del emisor A")
	assert.EqualValues(t, 2, got["count"])
}

func TestAPI_ListDocuments_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/documents", nil)
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_ListDocuments_InvalidLimit(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rw := env.doAuth(t, "GET", "/api/v1/documents?limit=no-es-un-numero", token, nil)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

// ── Middleware ───────────────────────────────────────────────────────────────────────────────

func TestAPI_RequestIDHeader(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/issuers/me", nil)
	assert.NotEmpty(t, rw.Header().Get("X-Request-ID"))
}

func TestAPI_UnknownRoute_404(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/no-existe", nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}
