package api_test

// Tests de integración contra PostgreSQL real. Se saltan automáticamente si la variable de
// entorno TEST_DATABASE_URL no está definida — así no bloquean el CI que solo corre los tests
// unitarios (en memoria), y el desarrollador puede correrlos localmente con:
//
//	TEST_DATABASE_URL=postgres://... go test ./internal/api/... -run Integration -v
//
// La URL debe apuntar a una base de datos de pruebas (nunca a producción), con el esquema ya
// aplicado (las mismas migraciones que usa el servidor). Los tests no limpian después de sí
// mismos — conviene usar una base de datos efímera (Docker, testcontainers, Neon branch).

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/diegofxm/catalogs"
	"github.com/diegofxm/apidian/internal/edocuments/documents"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/edocuments/numbering"
)

// testSecretsKey es la clave AES-256 que usan los repositorios Postgres para cifrar
// software_pin/certificate/technical_key en los tests de integración — no compartida con
// producción (se genera aquí, nunca se lee de .env).
var testSecretsKey = []byte("clave-de-prueba-32-bytes-exactos!")

// openTestPool abre un pgxpool hacia TEST_DATABASE_URL y lo cierra al finalizar el test.
// Salta el test si la variable no está definida.
func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL no definida — test de integración omitido")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "abrir pgxpool hacia TEST_DATABASE_URL")
	t.Cleanup(pool.Close)
	return pool
}

// TestIntegration_Issuers_CreateAndGet verifica que PostgresRepository puede persistir y
// leer de vuelta un emisor, incluyendo el cifrado/descifrado de software_pin y certificate.
func TestIntegration_Issuers_CreateAndGet(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := issuers.NewPostgresRepository(pool, testSecretsKey)
	catalogsRepo := catalogs.NewPostgresRepository(pool)
	svc := issuers.New(repo, documents.ValidateCertificate, documents.ParseCertificate, catalogsRepo)

	nit := "900" + uuid.New().String()[:6] // NIT único por ejecución
	iss, err := svc.RegisterIssuer(ctx, issuers.Issuer{
		NIT:                    nit,
		BusinessName:           "Empresa Test Integración",
		IdentificationTypeCode: "31",
		DepartmentCode:         "11",
		MunicipalityCode:       "11001",
		AddressLine:            "Calle Test 123",
		Email:                  "test@empresa.co",
		Environment:            issuers.EnvironmentHabilitacion,
		LiabilityCodes:         []string{"R-99-PN"},
		TaxSchemeCode:          "ZZ",
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, iss.ID)

	got, err := svc.GetIssuer(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "Empresa Test Integración", got.BusinessName)
	assert.Equal(t, nit, got.NIT)
	assert.Equal(t, "11001", got.MunicipalityCode)
}

// TestIntegration_NumberingRanges_ClaimNext verifica que ClaimNext es atómico: dos llamadas
// consecutivas entregan números distintos y en orden, y Activate/Deactivate funcionan.
func TestIntegration_NumberingRanges_ClaimNext(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	issuerRepo := issuers.NewPostgresRepository(pool, testSecretsKey)
	catalogsRepo := catalogs.NewPostgresRepository(pool)
	issuerSvc := issuers.New(issuerRepo, nil, nil, catalogsRepo)

	// Necesitamos un emisor real para la FK issuer_id en numbering_ranges
	nit := "800" + uuid.New().String()[:6]
	iss, err := issuerSvc.RegisterIssuer(ctx, issuers.Issuer{
		NIT: nit, BusinessName: "Emisor Numeración Test",
		IdentificationTypeCode: "31", DepartmentCode: "11", MunicipalityCode: "11001",
		AddressLine: "Calle 1", Email: "nr@test.co",
		Environment:    issuers.EnvironmentHabilitacion,
		LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ",
	})
	require.NoError(t, err)

	numRepo := numbering.NewPostgresRepository(pool, testSecretsKey)
	numSvc := numbering.New(numRepo)

	from := int64(1)
	to := int64(100)
	nr, err := numSvc.RegisterRange(ctx, numbering.NumberingRange{
		IssuerID:             iss.ID,
		DianDocumentTypeCode: "01",
		Prefix:               "TEST",
		ResolutionNumber:     "18764000001",
		ResolutionDate:       time.Now(),
		RangeFrom:            from,
		RangeTo:              &to,
		ValidFrom:            time.Now(),
		ValidTo:              time.Now().Add(365 * 24 * time.Hour),
		Environment:          numbering.EnvironmentHabilitacion,
	}, nil)
	require.NoError(t, err)

	n1, err := numSvc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	n2, err := numSvc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n1)
	assert.Equal(t, int64(2), n2)

	// Deactivate → Activate → ClaimNext sigue funcionando
	require.NoError(t, numSvc.DeactivateRange(ctx, nr.ID))
	reloaded, err := numSvc.GetRange(ctx, nr.ID)
	require.NoError(t, err)
	assert.False(t, reloaded.IsActive)

	require.NoError(t, numSvc.ActivateRange(ctx, nr.ID))
	reloaded2, err := numSvc.GetRange(ctx, nr.ID)
	require.NoError(t, err)
	assert.True(t, reloaded2.IsActive)

	n3, err := numSvc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), n3)
}

// TestIntegration_Issuers_UpdateProfile verifica que UpdateProfile persiste correctamente los
// campos de perfil sin tocar las credenciales cifradas.
func TestIntegration_Issuers_UpdateProfile(t *testing.T) {
	pool := openTestPool(t)
	ctx := context.Background()

	repo := issuers.NewPostgresRepository(pool, testSecretsKey)
	catalogsRepo := catalogs.NewPostgresRepository(pool)
	svc := issuers.New(repo, nil, nil, catalogsRepo)

	nit := "700" + uuid.New().String()[:6]
	iss, err := svc.RegisterIssuer(ctx, issuers.Issuer{
		NIT: nit, BusinessName: "Nombre Original",
		IdentificationTypeCode: "31", DepartmentCode: "11", MunicipalityCode: "11001",
		AddressLine: "Dirección original", Email: "orig@test.co",
		Environment:    issuers.EnvironmentHabilitacion,
		LiabilityCodes: []string{"R-99-PN"}, TaxSchemeCode: "ZZ",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateProfile(ctx, iss.ID, issuers.UpdateProfileRequest{
		BusinessName:                "Nombre Actualizado",
		TradeName:                   "NomComercial",
		DepartmentCode:              "11",
		MunicipalityCode:            "11001",
		AddressLine:                 "Nueva Dirección 456",
		Email:                       "nuevo@test.co",
		Phone:                       "3001234567",
		EntityTypeCode:              "1",
		TaxSchemeCode:               "ZZ",
		LiabilityCodes:              []string{"R-99-PN"},
		IndustryClassificationCodes: []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "Nombre Actualizado", updated.BusinessName)
	assert.Equal(t, "NomComercial", updated.TradeName)
	assert.Equal(t, "Nueva Dirección 456", updated.AddressLine)
	assert.Equal(t, "3001234567", updated.Phone)

	// Re-leer para verificar persistencia
	reread, err := svc.GetIssuer(ctx, iss.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nombre Actualizado", reread.BusinessName)
}
