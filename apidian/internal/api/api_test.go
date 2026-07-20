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

	"github.com/diegofxm/apidian/internal/api"
	"github.com/diegofxm/apidian/internal/auth"
	"github.com/diegofxm/apidian/internal/catalogs"
	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/products"
	"github.com/diegofxm/apidian/internal/vendors"
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
		Subject:      pkix.Name{CommonName: "apidian test"},
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

// newTestEnv usa "*" como orígenes CORS permitidos — ninguno de estos tests manda un header
// Origin, así que esto no cambia ningún comportamiento existente; ver newTestEnvWithOrigins
// para los tests que sí necesitan probar CORS con una lista explícita.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWithOrigins(t, []string{"*"})
}

func newTestEnvWithOrigins(t *testing.T, allowedOrigins []string) *testEnv {
	t.Helper()
	log := zap.NewNop()

	catalogsRepo := catalogs.NewMemoryRepository()
	issuerSvc := issuers.New(issuers.NewMemoryRepository(), documents.ValidateCertificate, documents.ParseCertificate, catalogsRepo)
	numberingSvc := numbering.New(numbering.NewMemoryRepository())
	customersSvc := customers.New(customers.NewMemoryRepository(), catalogsRepo)
	productsSvc := products.New(products.NewMemoryRepository(), catalogsRepo)
	vendorsSvc := vendors.New(vendors.NewMemoryRepository(), catalogsRepo)
	docsSvc := documents.New(documents.NewMemoryRepository(), issuerSvc, numberingSvc, customersSvc, catalogsRepo, email.NewSMTPSender(email.Config{})).WithVendors(vendorsSvc)
	tokens := auth.NewTokenIssuer([]byte("clave-de-prueba-no-usar-en-produccion"))
	authSvc := auth.New(auth.NewMemoryRepository(), issuerSvc, tokens)

	a := api.NewFromServices(log, issuerSvc, numberingSvc, docsSvc, authSvc, tokens, customersSvc, productsSvc, vendorsSvc, nil, nil, nil, catalogsRepo, allowedOrigins)

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

// registerRequest construye un payload de POST /api/v1/auth/register completo — desde la
// Fase 9.32 ya NO incluye una empresa (registro desacoplado, ver sección 9.32). email permite
// variar el correo entre llamadas — el mismo correo no puede registrarse dos veces.
func registerRequest(email string) map[string]any {
	return map[string]any{
		"email":    email,
		"password": "contraseña-segura",
		"name":     "Admin de Prueba",
	}
}

// createIssuerBody construye un payload de POST /api/v1/issuers completo. El NIT se genera
// único por llamada: dos empresas no pueden compartir NIT.
func createIssuerBody(t *testing.T) map[string]any {
	nit := fmt.Sprintf("9%011d", time.Now().UnixNano()%100000000000)
	return map[string]any{
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
	}
}

// createIssuerBodyWithoutCredentials es createIssuerBody sin software_id/software_pin/
// certificate_base64/certificate_password — confirma que crear una empresa ya no los exige
// (ver docs/apidian-architecture.md sección 9.25): solo los datos que la DIAN pide del
// emisor mismo.
func createIssuerBodyWithoutCredentials(t *testing.T) map[string]any {
	body := createIssuerBody(t)
	delete(body, "software_id")
	delete(body, "software_pin")
	delete(body, "certificate_base64")
	delete(body, "certificate_password")
	return body
}

// registerAndCreateIssuer registra un usuario nuevo y le crea una empresa — devuelve
// (issuerID, token) con el token YA activo en esa empresa (ver auth.Service.CreateIssuerForUser).
func registerAndCreateIssuer(t *testing.T, env *testEnv, email string, issuerBody map[string]any) (string, string) {
	t.Helper()
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest(email))
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var registered map[string]any
	decode(t, rw, &registered)
	registerToken := registered["token"].(string)

	rw = env.doAuth(t, "POST", "/api/v1/issuers", registerToken, issuerBody)
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var created map[string]any
	decode(t, rw, &created)
	issuerID := created["issuer"].(map[string]any)["id"].(string)
	token := created["token"].(string)
	return issuerID, token
}

// registerTestIssuer registra un usuario y le crea una empresa con credenciales completas —
// devuelve (issuerID, token).
func registerTestIssuer(t *testing.T, env *testEnv) (string, string) {
	t.Helper()
	email := fmt.Sprintf("admin-%s@empresa.test", uuid.New().String())
	return registerAndCreateIssuer(t, env, email, createIssuerBody(t))
}

func TestAPI_Register_OK(t *testing.T) {
	env := newTestEnv(t)

	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("admin@empresa.test"))

	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotEmpty(t, got["token"])
	assert.Equal(t, "admin@empresa.test", got["user"].(map[string]any)["email"])
	assert.NotContains(t, got, "issuer", "un usuario recién registrado no tiene ninguna empresa todavía")
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
	req := registerRequest("admin@empresa.test")
	req["name"] = ""
	rw := env.do(t, "POST", "/api/v1/auth/register", req)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_Register_DuplicateEmail(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("admin@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)

	rw = env.do(t, "POST", "/api/v1/auth/register", registerRequest("admin@empresa.test"))
	assert.Equal(t, http.StatusConflict, rw.Code)
}

func TestAPI_Login_OK(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("admin@empresa.test"))
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

// TestAPI_Login_AutoSelectsSingleIssuer confirma el comportamiento elegido para el caso
// normal de hoy: con una sola empresa vinculada, el login ya viene con ella activa, sin un
// paso extra de selección (ver sección 9.32).
func TestAPI_Login_AutoSelectsSingleIssuer(t *testing.T) {
	env := newTestEnv(t)
	email := "auto-select@empresa.test"
	registerAndCreateIssuer(t, env, email, createIssuerBody(t))

	rw := env.do(t, "POST", "/api/v1/auth/login", map[string]any{"email": email, "password": "contraseña-segura"})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotEmpty(t, got["issuer"], "con una sola empresa vinculada, el login debe autoseleccionarla")
}

func TestAPI_Login_WrongPassword(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("admin@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)

	rw = env.do(t, "POST", "/api/v1/auth/login", map[string]any{
		"email":    "admin@empresa.test",
		"password": "contraseña-equivocada",
	})
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

// ── Empresas (multi-empresa, sección 9.32) ──────────────────────────────────────────────────

func TestAPI_CreateIssuer_OK(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("crea-empresa@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)

	rw = env.doAuth(t, "POST", "/api/v1/issuers", registered["token"].(string), createIssuerBody(t))
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotEmpty(t, got["token"], "debe venir un token nuevo con la empresa activa")
	assert.NotEmpty(t, got["issuer"].(map[string]any)["nit"])
}

func TestAPI_CreateIssuer_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/issuers", createIssuerBody(t))
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_CreateIssuer_WithoutCredentials_OK(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("sin-credenciales-empresa@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)

	rw = env.doAuth(t, "POST", "/api/v1/issuers", registered["token"].(string), createIssuerBodyWithoutCredentials(t))
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
}

func TestAPI_ListMyIssuers(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("lista-empresas@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)
	registerToken := registered["token"].(string)

	rw = env.doAuth(t, "GET", "/api/v1/issuers", registerToken, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.EqualValues(t, 0, got["count"])

	createRw := env.doAuth(t, "POST", "/api/v1/issuers", registerToken, createIssuerBody(t))
	require.Equal(t, http.StatusCreated, createRw.Code)

	rw = env.doAuth(t, "GET", "/api/v1/issuers", registerToken, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	decode(t, rw, &got)
	assert.EqualValues(t, 1, got["count"])
}

func TestAPI_SelectIssuer_OK(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("selecciona-empresa@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)
	registerToken := registered["token"].(string)

	createRw := env.doAuth(t, "POST", "/api/v1/issuers", registerToken, createIssuerBody(t))
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)
	issuerID := created["issuer"].(map[string]any)["id"].(string)

	rw = env.doAuth(t, "POST", "/api/v1/issuers/"+issuerID+"/select", registerToken, nil)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, issuerID, got["issuer"].(map[string]any)["id"])
}

// TestAPI_SelectIssuer_AccessDenied confirma el aislamiento entre tenants para el nuevo
// endpoint: un usuario no puede activar una empresa a la que nunca se vinculó.
func TestAPI_SelectIssuer_AccessDenied(t *testing.T) {
	env := newTestEnv(t)
	otherIssuerID, _ := registerTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("sin-acceso@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)

	rw = env.doAuth(t, "POST", "/api/v1/issuers/"+otherIssuerID+"/select", registered["token"].(string), nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

// TestAPI_RequireTenant_NoActiveCompany confirma que las rutas que necesitan una empresa
// activa fallan con un mensaje claro (409) cuando el usuario todavía no tiene ninguna — en vez
// de devolver listas vacías silenciosamente o fallar de forma confusa más abajo.
func TestAPI_RequireTenant_NoActiveCompany(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/auth/register", registerRequest("sin-empresa-activa@empresa.test"))
	require.Equal(t, http.StatusCreated, rw.Code)
	var registered map[string]any
	decode(t, rw, &registered)

	rw = env.doAuth(t, "GET", "/api/v1/issuers/me", registered["token"].(string), nil)
	assert.Equal(t, http.StatusConflict, rw.Code)
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

func TestAPI_UpdateMyIssuer_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "PUT", "/api/v1/issuers/me", token, map[string]any{
		"software_id":          "software-id-nuevo",
		"software_pin":         "54321",
		"certificate_base64":   selfSignedP12Base64(t),
		"certificate_password": "clave-de-prueba",
	})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.NotContains(t, got, "software_pin", "el secreto nunca debe salir en la respuesta")
}

func TestAPI_UpdateMyIssuer_Partial(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	// Solo software_id — software_pin/certificado ya cargados en el registro no deben tocarse.
	rw := env.doAuth(t, "PUT", "/api/v1/issuers/me", token, map[string]any{
		"software_id": "software-id-actualizado",
	})
	assert.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
}

func TestAPI_UpdateMyIssuer_EmptyValueRejected(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "PUT", "/api/v1/issuers/me", token, map[string]any{
		"software_id": "",
	})
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_UpdateMyIssuer_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "PUT", "/api/v1/issuers/me", map[string]any{"software_id": "x"})
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

// TestAPI_ConfirmDocument_IssuerNotReady es el cierre del ciclo de configuración gradual: un
// emisor registrado SIN software/certificado puede crear borradores (no se exige nada
// todavía) pero no confirmarlos — error de dominio claro, no un fallo de bajo nivel al
// intentar parsear un certificado vacío.
func TestAPI_ConfirmDocument_IssuerNotReady(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerAndCreateIssuer(t, env, "sin-credenciales@empresa.test", createIssuerBodyWithoutCredentials(t))

	rangeID := createTestRange(t, env, token, "01", "SETP")
	draft := createDraftInvoice(t, env, token, rangeID)

	confirmRw := confirmDocument(env, t, token, draft["id"].(string))
	assert.Equal(t, http.StatusUnprocessableEntity, confirmRw.Code, confirmRw.Body.String())
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
		"test_set_id":             "653bf9d9-b2b1-44ae-a66d-3b9cdc4271c3",
	})

	require.Equal(t, http.StatusCreated, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "SETP", got["prefix"])
	assert.Equal(t, "653bf9d9-b2b1-44ae-a66d-3b9cdc4271c3", got["test_set_id"])
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

// testPaymentMeans — cac:PaymentMeans es obligatorio (1..N) para Invoice/CreditNote/DebitNote
// según el Anexo Técnico (ver docs/apidian-architecture.md sección 9.30); sin esto la DIAN
// rechaza con "errores en campos mandatorios".
func testPaymentMeans() []map[string]any {
	return []map[string]any{{"code": "1", "payment_method_code": "10"}}
}

// createDraftInvoice crea un borrador de Factura (sin reclamar número, ver
// docs/apidian-architecture.md sección 9.25) y devuelve el body decodificado.
func createDraftInvoice(t *testing.T, env *testEnv, token, rangeID string) map[string]any {
	t.Helper()
	rw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
	})
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	return got
}

func confirmDocument(env *testEnv, t *testing.T, token, id string) *httptest.ResponseRecorder {
	t.Helper()
	return env.doAuth(t, "POST", "/api/v1/documents/"+id+"/confirm", token, nil)
}

// issueInvoiceViaAPI crea el borrador y lo confirma de una — para los tests a los que solo
// les importa el resultado final ya firmado, no el paso intermedio de borrador.
func issueInvoiceViaAPI(t *testing.T, env *testEnv, token, rangeID string) map[string]any {
	t.Helper()
	draft := createDraftInvoice(t, env, token, rangeID)
	rw := confirmDocument(env, t, token, draft["id"].(string))
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	return got
}

func TestAPI_CreateInvoiceDraft_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	draft := createDraftInvoice(t, env, token, rangeID)
	assert.Equal(t, "draft", draft["status"])
	assert.Empty(t, draft["number"], "un borrador no reclama número")
	assert.Empty(t, draft["document_key"], "un borrador no tiene CUFE todavía")
	assert.Empty(t, draft["signed_xml"], "un borrador no está firmado todavía")

	// El contenido SÍ debe verse desde el borrador — un frontend real necesita mostrar
	// "factura para X por $Y" sin tener que volver a pedirle los datos al usuario (ver
	// docs/apidian-architecture.md sección 9.28).
	customer, ok := draft["customer"].(map[string]any)
	require.True(t, ok, "customer debe venir en la respuesta")
	assert.Equal(t, "Consumidor Final", customer["name"])
	lines, ok := draft["lines"].([]any)
	require.True(t, ok, "lines debe venir en la respuesta")
	assert.Len(t, lines, 1)
	totals, ok := draft["totals"].(map[string]any)
	require.True(t, ok, "totals debe venir en la respuesta")
	assert.EqualValues(t, 10000, totals["line_extension_cents"])
}

func TestAPI_ConfirmDocument_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	got := issueInvoiceViaAPI(t, env, token, rangeID)
	assert.Equal(t, "SETP", got["prefix"])
	// Emisor y rango están en producción → intenta enviar a soap.ProduccionURL → no hay red real
	// en httptest → send_error es el estado final esperado. El documento SÍ queda construido
	// (tiene CUFE, XML firmado y número reservado).
	assert.Equal(t, "send_error", got["status"])
	assert.NotEmpty(t, got["document_key"])
	// signed_xml ya no viaja en el JSON — se descarga por GET /documents/{id}/xml.
	assert.Nil(t, got["signed_xml"], "signed_xml no debe aparecer en la respuesta JSON")

	customer, ok := got["customer"].(map[string]any)
	require.True(t, ok, "customer debe seguir presente después de confirmar")
	assert.Equal(t, "Consumidor Final", customer["name"])
}

func TestAPI_GetDocumentXML_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	doc := issueInvoiceViaAPI(t, env, token, rangeID)
	id := doc["id"].(string)

	rw := env.doAuth(t, "GET", "/api/v1/documents/"+id+"/xml", token, nil)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	assert.Equal(t, "application/xml", rw.Header().Get("Content-Type"))
	assert.Contains(t, rw.Body.String(), "<ds:Signature")
}

func TestAPI_GetDocumentXML_Draft(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	draft := createDraftInvoice(t, env, token, rangeID)
	rw := env.doAuth(t, "GET", "/api/v1/documents/"+draft["id"].(string)+"/xml", token, nil)
	assert.Equal(t, http.StatusConflict, rw.Code, "un borrador no tiene XML todavía")
}

func TestAPI_ConfirmDocument_NotDraft(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	doc := issueInvoiceViaAPI(t, env, token, rangeID)

	rw := confirmDocument(env, t, token, doc["id"].(string))
	assert.Equal(t, http.StatusConflict, rw.Code, "confirmar dos veces no debe gastar un segundo número")
}

func TestAPI_ConfirmDocument_NotFound(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := confirmDocument(env, t, token, uuid.New().String())
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_ConfirmDocument_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	draft := createDraftInvoice(t, env, tokenA, rangeID)

	// El emisor B no debe poder confirmar un borrador del emisor A — mismo 404 que si no existiera.
	rw := confirmDocument(env, t, tokenB, draft["id"].(string))
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_UpdateInvoiceDraft_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	draft := createDraftInvoice(t, env, token, rangeID)

	lines := testLines()
	lines[0]["description"] = "Servicio corregido"
	rw := env.doAuth(t, "PUT", "/api/v1/invoices/"+draft["id"].(string), token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              lines,
		"payment_means":      testPaymentMeans(),
	})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "draft", got["status"])
}

func TestAPI_UpdateInvoiceDraft_NotDraft(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	doc := issueInvoiceViaAPI(t, env, token, rangeID)

	rw := env.doAuth(t, "PUT", "/api/v1/invoices/"+doc["id"].(string), token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
	})
	assert.Equal(t, http.StatusConflict, rw.Code, "un documento ya confirmado es inmutable")
}

func TestAPI_UpdateInvoiceDraft_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	draft := createDraftInvoice(t, env, tokenA, rangeID)

	rw := env.doAuth(t, "PUT", "/api/v1/invoices/"+draft["id"].(string), tokenB, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
	})
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_DeleteDraft_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	draft := createDraftInvoice(t, env, token, rangeID)

	rw := env.doAuth(t, "DELETE", "/api/v1/documents/"+draft["id"].(string), token, nil)
	assert.Equal(t, http.StatusNoContent, rw.Code)

	rw = env.doAuth(t, "GET", "/api/v1/documents/"+draft["id"].(string), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_DeleteDraft_NotDraft(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	doc := issueInvoiceViaAPI(t, env, token, rangeID)

	rw := env.doAuth(t, "DELETE", "/api/v1/documents/"+doc["id"].(string), token, nil)
	assert.Equal(t, http.StatusConflict, rw.Code, "un documento ya confirmado nunca se borra")
}

func TestAPI_DeleteDraft_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")

	draft := createDraftInvoice(t, env, tokenA, rangeID)

	rw := env.doAuth(t, "DELETE", "/api/v1/documents/"+draft["id"].(string), tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_IssueInvoice_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/invoices", map[string]any{
		"numbering_range_id": uuid.New().String(),
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
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
		"payment_means":      testPaymentMeans(),
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
		"payment_means":      testPaymentMeans(),
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rw.Code)
}

func createTestCustomer(t *testing.T, env *testEnv, token string) string {
	t.Helper()
	rw := env.doAuth(t, "POST", "/api/v1/customers", token, testCustomer())
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	return got["id"].(string)
}

func TestAPI_IssueInvoice_WithCustomerID_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")
	customerID := createTestCustomer(t, env, token)

	body := testCustomer()
	rw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           body,
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
		"customer_id":        customerID,
	})

	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, customerID, got["customer_id"])
}

func TestAPI_IssueInvoice_OtherTenantCustomerID(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, tokenA, "01", "SETP")
	customerIDFromB := createTestCustomer(t, env, tokenB)

	// El emisor A no debe poder emitir referenciando el cliente del emisor B.
	rw := env.doAuth(t, "POST", "/api/v1/invoices", tokenA, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
		"customer_id":        customerIDFromB,
	})

	assert.Equal(t, http.StatusUnprocessableEntity, rw.Code)
}

func TestAPI_IssueInvoice_InvalidCustomerID(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rangeID := createTestRange(t, env, token, "01", "SETP")

	rw := env.doAuth(t, "POST", "/api/v1/invoices", token, map[string]any{
		"numbering_range_id": rangeID,
		"customer":           testCustomer(),
		"lines":              testLines(),
		"payment_means":      testPaymentMeans(),
		"customer_id":        "no-es-un-uuid",
	})

	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_IssueCreditNote_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	invoiceRangeID := createTestRange(t, env, token, "01", "SETP")
	cnRangeID := createTestRange(t, env, token, "91", "SETPNC")

	inv := issueInvoiceViaAPI(t, env, token, invoiceRangeID)

	rw := env.doAuth(t, "POST", "/api/v1/credit-notes", token, map[string]any{
		"numbering_range_id":    cnRangeID,
		"customer":              testCustomer(),
		"lines":                 testLines(),
		"payment_means":         testPaymentMeans(),
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
	// Un borrador no reclama prefix/number todavía (ver model.go) — eso solo se llena al
	// confirmar. La identidad de un borrador es el numbering_range_id, no el prefix.
	assert.Equal(t, cnRangeID, got["numbering_range_id"])
	assert.Equal(t, "91", got["dian_document_type_code"])
	assert.Equal(t, "draft", got["status"], "el borrador de la nota tampoco se confirma solo")

	confirmRw := confirmDocument(env, t, token, got["id"].(string))
	require.Equal(t, http.StatusOK, confirmRw.Code, confirmRw.Body.String())
	var confirmed map[string]any
	decode(t, confirmRw, &confirmed)
	assert.Equal(t, "SETPNC", confirmed["prefix"])
	// Mismo criterio que TestAPI_ConfirmDocument_OK: rango en producción → intenta enviar →
	// sin red real en httptest → send_error esperado.
	assert.Equal(t, "send_error", confirmed["status"])
}

func TestAPI_IssueCreditNote_MissingBillingReference(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	cnRangeID := createTestRange(t, env, token, "91", "SETPNC")

	rw := env.doAuth(t, "POST", "/api/v1/credit-notes", token, map[string]any{
		"numbering_range_id":    cnRangeID,
		"customer":              testCustomer(),
		"lines":                 testLines(),
		"payment_means":         testPaymentMeans(),
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

	created := issueInvoiceViaAPI(t, env, token, rangeID)

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

	created := createDraftInvoice(t, env, tokenA, rangeID)

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

	createDraftInvoice(t, env, tokenA, rangeA)
	createDraftInvoice(t, env, tokenA, rangeA)
	createDraftInvoice(t, env, tokenB, rangeB)

	rw := env.doAuth(t, "GET", "/api/v1/documents", tokenA, nil)
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

// ── Customers ────────────────────────────────────────────────────────────────────────────────

func TestAPI_CreateCustomer_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "POST", "/api/v1/customers", token, testCustomer())
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "Consumidor Final", got["name"])
	assert.NotEmpty(t, got["id"])
}

func TestAPI_CreateCustomer_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/customers", testCustomer())
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_CreateCustomer_MissingName(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	c := testCustomer()
	delete(c, "name")
	rw := env.doAuth(t, "POST", "/api/v1/customers", token, c)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_GetCustomer_NotFound(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)
	rw := env.doAuth(t, "GET", "/api/v1/customers/"+uuid.New().String(), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetCustomer_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/customers", tokenA, testCustomer())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "GET", "/api/v1/customers/"+created["id"].(string), tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_ListCustomers_OnlyOwnTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	require.Equal(t, http.StatusCreated, env.doAuth(t, "POST", "/api/v1/customers", tokenA, testCustomer()).Code)
	require.Equal(t, http.StatusCreated, env.doAuth(t, "POST", "/api/v1/customers", tokenA, testCustomer()).Code)
	require.Equal(t, http.StatusCreated, env.doAuth(t, "POST", "/api/v1/customers", tokenB, testCustomer()).Code)

	rw := env.doAuth(t, "GET", "/api/v1/customers", tokenA, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Len(t, got["customers"].([]any), 2)
}

func TestAPI_UpdateCustomer_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/customers", token, testCustomer())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	updated := testCustomer()
	updated["name"] = "Nombre Nuevo"
	rw := env.doAuth(t, "PUT", "/api/v1/customers/"+created["id"].(string), token, updated)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "Nombre Nuevo", got["name"])
}

func TestAPI_UpdateCustomer_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/customers", tokenA, testCustomer())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "PUT", "/api/v1/customers/"+created["id"].(string), tokenB, testCustomer())
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_DeleteCustomer_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/customers", token, testCustomer())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "DELETE", "/api/v1/customers/"+created["id"].(string), token, nil)
	assert.Equal(t, http.StatusNoContent, rw.Code)

	rw = env.doAuth(t, "GET", "/api/v1/customers/"+created["id"].(string), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_DeleteCustomer_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/customers", tokenA, testCustomer())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "DELETE", "/api/v1/customers/"+created["id"].(string), tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

// ── Autorregistro público (patrón QR/D1, sección 9.41) ─────────────────────────────────────────

func TestAPI_GetPublicIssuer_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID, _ := registerTestIssuer(t, env)

	rw := env.do(t, "GET", "/api/v1/public/issuers/"+issuerID, nil)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "Empresa de Prueba S.A.S.", got["business_name"])
	assert.Nil(t, got["has_software_credentials"], "el DTO público nunca debe traer estos campos")
}

func TestAPI_GetPublicIssuer_NotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/public/issuers/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

// tinyPNGBase64 es un PNG 1x1 transparente válido — mismo criterio que selfSignedP12Base64,
// el contenido real no importa para estas pruebas, solo que sea un archivo de imagen real.
const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

func TestAPI_GetPublicIssuer_HasLogo(t *testing.T) {
	env := newTestEnv(t)
	issuerID, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "PUT", "/api/v1/issuers/me", token, map[string]any{
		"logo_base64":       tinyPNGBase64,
		"logo_content_type": "png",
	})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = env.do(t, "GET", "/api/v1/public/issuers/"+issuerID, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, true, got["has_logo"])
}

func TestAPI_GetPublicIssuerLogo_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "PUT", "/api/v1/issuers/me", token, map[string]any{
		"logo_base64":       tinyPNGBase64,
		"logo_content_type": "png",
	})
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())

	rw = env.do(t, "GET", "/api/v1/public/issuers/"+issuerID+"/logo", nil)
	require.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "image/png", rw.Header().Get("Content-Type"))
	assert.NotEmpty(t, rw.Body.Bytes())
}

func TestAPI_GetPublicIssuerLogo_NotConfigured(t *testing.T) {
	env := newTestEnv(t)
	issuerID, _ := registerTestIssuer(t, env)

	rw := env.do(t, "GET", "/api/v1/public/issuers/"+issuerID+"/logo", nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_GetPublicIssuerLogo_IssuerNotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "GET", "/api/v1/public/issuers/"+uuid.New().String()+"/logo", nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_CreatePublicCustomer_OK(t *testing.T) {
	env := newTestEnv(t)
	issuerID, token := registerTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/public/issuers/"+issuerID+"/customers", map[string]any{
		"identification": map[string]any{"number": "1122334455", "type_code": "13"},
		"name":           "Cliente de Mostrador",
		"email":          "cliente@correo.test",
	})
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "Cliente de Mostrador", got["name"])

	// El cliente debe quedar guardado de verdad en el catálogo del emisor — no es solo un eco.
	listRw := env.doAuth(t, "GET", "/api/v1/customers", token, nil)
	require.Equal(t, http.StatusOK, listRw.Code)
	var list map[string]any
	decode(t, listRw, &list)
	assert.EqualValues(t, 1, list["count"])
}

func TestAPI_CreatePublicCustomer_IssuerNotFound(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/public/issuers/"+uuid.New().String()+"/customers", map[string]any{
		"identification": map[string]any{"number": "1122334455", "type_code": "13"},
		"name":           "Cliente de Mostrador",
	})
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_CreatePublicCustomer_MissingName(t *testing.T) {
	env := newTestEnv(t)
	issuerID, _ := registerTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/public/issuers/"+issuerID+"/customers", map[string]any{
		"identification": map[string]any{"number": "1122334455", "type_code": "13"},
	})
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

// TestAPI_CreatePublicCustomer_DerivesNITCheckDigit confirma que el dígito de verificación se
// deriva igual que en el flujo autenticado (ver internal/nit) — el visitante anónimo no manda
// uno y no hace falta: 6382356 es el caso real verificado (dígito 7).
func TestAPI_CreatePublicCustomer_DerivesNITCheckDigit(t *testing.T) {
	env := newTestEnv(t)
	issuerID, token := registerTestIssuer(t, env)

	rw := env.do(t, "POST", "/api/v1/public/issuers/"+issuerID+"/customers", map[string]any{
		"identification": map[string]any{"number": "6382356", "type_code": "31"},
		"name":           "Empresa Cliente S.A.S.",
	})
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())

	listRw := env.doAuth(t, "GET", "/api/v1/customers", token, nil)
	require.Equal(t, http.StatusOK, listRw.Code)
	var list map[string]any
	decode(t, listRw, &list)
	customers := list["customers"].([]any)
	require.Len(t, customers, 1)
	identification := customers[0].(map[string]any)["identification"].(map[string]any)
	assert.Equal(t, "7", identification["verification_code"])
}

// ── Products ─────────────────────────────────────────────────────────────────────────────────

func testProduct() map[string]any {
	return map[string]any{
		"description":      "Servicio de consultoría",
		"unit_code":        "94",
		"unit_price_cents": 100000,
		"item_code":        "SVC-001",
		"item_type_code":   "999",
		"item_type_name":   "Estándar de adopción del contribuyente",
		"tax_type_code":    "01",
		"tax_type_name":    "IVA",
		"tax_percent":      19,
	}
}

func TestAPI_CreateProduct_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	rw := env.doAuth(t, "POST", "/api/v1/products", token, testProduct())
	require.Equal(t, http.StatusCreated, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.Equal(t, "Servicio de consultoría", got["description"])
	assert.NotEmpty(t, got["id"])
}

func TestAPI_CreateProduct_NoToken(t *testing.T) {
	env := newTestEnv(t)
	rw := env.do(t, "POST", "/api/v1/products", testProduct())
	assert.Equal(t, http.StatusUnauthorized, rw.Code)
}

func TestAPI_CreateProduct_MissingDescription(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	p := testProduct()
	delete(p, "description")
	rw := env.doAuth(t, "POST", "/api/v1/products", token, p)
	assert.Equal(t, http.StatusBadRequest, rw.Code)
}

func TestAPI_ListProducts_OnlyOwnTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	require.Equal(t, http.StatusCreated, env.doAuth(t, "POST", "/api/v1/products", tokenA, testProduct()).Code)
	require.Equal(t, http.StatusCreated, env.doAuth(t, "POST", "/api/v1/products", tokenB, testProduct()).Code)

	rw := env.doAuth(t, "GET", "/api/v1/products", tokenA, nil)
	require.Equal(t, http.StatusOK, rw.Code)
	var got map[string]any
	decode(t, rw, &got)
	assert.Len(t, got["products"].([]any), 1)
}

func TestAPI_UpdateProduct_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/products", token, testProduct())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	updated := testProduct()
	updated["unit_price_cents"] = 200000
	rw := env.doAuth(t, "PUT", "/api/v1/products/"+created["id"].(string), token, updated)
	require.Equal(t, http.StatusOK, rw.Code, rw.Body.String())
	var got map[string]any
	decode(t, rw, &got)
	assert.EqualValues(t, 200000, got["unit_price_cents"])
}

func TestAPI_DeleteProduct_OK(t *testing.T) {
	env := newTestEnv(t)
	_, token := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/products", token, testProduct())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "DELETE", "/api/v1/products/"+created["id"].(string), token, nil)
	assert.Equal(t, http.StatusNoContent, rw.Code)

	rw = env.doAuth(t, "GET", "/api/v1/products/"+created["id"].(string), token, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
}

func TestAPI_DeleteProduct_OtherTenant(t *testing.T) {
	env := newTestEnv(t)
	_, tokenA := registerTestIssuer(t, env)
	_, tokenB := registerTestIssuer(t, env)

	createRw := env.doAuth(t, "POST", "/api/v1/products", tokenA, testProduct())
	require.Equal(t, http.StatusCreated, createRw.Code)
	var created map[string]any
	decode(t, createRw, &created)

	rw := env.doAuth(t, "DELETE", "/api/v1/products/"+created["id"].(string), tokenB, nil)
	assert.Equal(t, http.StatusNotFound, rw.Code)
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

// ── CORS ─────────────────────────────────────────────────────────────────────────────────────

// TestAPI_CORS_PreflightAllowedOrigin confirma que un preflight OPTIONS desde un origen
// permitido se responde directamente (204), sin llegar al mux — que no tiene ninguna ruta
// OPTIONS registrada y respondería 405/404 si el preflight no se cortara antes.
func TestAPI_CORS_PreflightAllowedOrigin(t *testing.T) {
	env := newTestEnvWithOrigins(t, []string{"http://localhost:5500"})

	req := httptest.NewRequest("OPTIONS", "/api/v1/invoices", nil)
	req.Header.Set("Origin", "http://localhost:5500")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, "http://localhost:5500", rw.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rw.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, rw.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

// TestAPI_CORS_PreflightDisallowedOrigin confirma que un origen NO permitido nunca recibe
// Access-Control-Allow-Origin — el navegador lo bloqueará del lado del cliente.
func TestAPI_CORS_PreflightDisallowedOrigin(t *testing.T) {
	env := newTestEnvWithOrigins(t, []string{"http://localhost:5500"})

	req := httptest.NewRequest("OPTIONS", "/api/v1/invoices", nil)
	req.Header.Set("Origin", "http://evil.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)

	assert.Empty(t, rw.Header().Get("Access-Control-Allow-Origin"))
}

// TestAPI_CORS_ActualRequestEchoesOrigin confirma que una petición real (no preflight) desde
// un origen permitido recibe el header reflejado exactamente (nunca "*" cuando la lista es
// explícita) más Vary: Origin.
func TestAPI_CORS_ActualRequestEchoesOrigin(t *testing.T) {
	env := newTestEnvWithOrigins(t, []string{"http://localhost:5500"})
	_, token := registerTestIssuer(t, env)

	req := httptest.NewRequest("GET", "/api/v1/issuers/me", nil)
	req.Header.Set("Origin", "http://localhost:5500")
	req.Header.Set("Authorization", "Bearer "+token)
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "http://localhost:5500", rw.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rw.Header().Get("Vary"))
}

// TestAPI_CORS_WildcardAllowsAnyOrigin confirma que configurar "*" (solo para desarrollo
// local) acepta cualquier origen, con el literal "*" — no el origen reflejado.
func TestAPI_CORS_WildcardAllowsAnyOrigin(t *testing.T) {
	env := newTestEnvWithOrigins(t, []string{"*"})

	req := httptest.NewRequest("OPTIONS", "/api/v1/invoices", nil)
	req.Header.Set("Origin", "http://cualquier-cosa.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rw := httptest.NewRecorder()
	env.handler.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusNoContent, rw.Code)
	assert.Equal(t, "*", rw.Header().Get("Access-Control-Allow-Origin"))
}
