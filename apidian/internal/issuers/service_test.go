package issuers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCatalogPort acepta cualquier código por defecto — TestRegisterIssuer_Validations pone
// valid en false explícitamente para probar el rechazo de un código inválido.
type fakeCatalogPort struct {
	valid bool
}

func (f *fakeCatalogPort) IsValidLiabilityCode(_ context.Context, _ string) (bool, error) {
	return f.valid, nil
}

// GetTaxTypeName replica el subconjunto de tax_types que usan estos tests — "ZZ" es el
// default real que applyDefaults aplica cuando el emisor no manda tax_scheme_code.
func (f *fakeCatalogPort) GetTaxTypeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"ZZ": "No aplica", "01": "IVA"}
	name, ok := names[code]
	return name, ok, nil
}

// newService usa un validador permisivo (nunca rechaza) — estos tests prueban lógica de
// dominio (NIT duplicado, campos vacíos, actualización parcial...), no el parseo real de un
// .p12. El parseo real se prueba aparte en TestUpdateIssuer_InvalidCertificate con un
// validador doble que sí falla, y en internal/api (vía documents.ValidateCertificate real).
func newService() *issuers.Service {
	return issuers.New(issuers.NewMemoryRepository(), func([]byte, string) error { return nil }, nil, &fakeCatalogPort{valid: true})
}

func validIssuer() issuers.Issuer {
	return issuers.Issuer{
		NIT:                    "900373076",
		CheckDigit:             "1",
		BusinessName:           "Empresa de Prueba S.A.S.",
		IdentificationTypeCode: "31",
		DepartmentCode:         "11",
		MunicipalityCode:       "11001",
		AddressLine:            "Calle 1 # 2-3",
		Email:                  "facturacion@empresa.test",
		Environment:            issuers.EnvironmentHabilitacion,
		SoftwareID:             "software-id-de-prueba",
		SoftwarePIN:            "12345",
		Certificate:            []byte("contenido-p12-de-prueba"),
		CertificatePassword:    "clave-del-certificado",
	}
}

func TestRegisterIssuer_OK(t *testing.T) {
	svc := newService()
	iss, err := svc.RegisterIssuer(context.Background(), validIssuer())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, iss.ID)
	assert.True(t, iss.IsActive)
	assert.Equal(t, "1", iss.EntityTypeCode, "NIT (31) -> Persona Jurídica, default validado contra la DIAN real")
	assert.Equal(t, "ZZ", iss.TaxSchemeCode)
	assert.Equal(t, "No aplica", iss.TaxSchemeName)
	assert.Equal(t, []string{"R-99-PN"}, iss.LiabilityCodes, "sin esto la DIAN rechaza el documento por campo mandatorio faltante")
	assert.False(t, iss.CreatedAt.IsZero())
}

// TestRegisterIssuer_NaturalPerson_EntityTypeCode confirma el fix de un rechazo real de la
// DIAN (StatusCode 99, "errores en campos mandatorios", 2026-06-23): un emisor identificado
// con cédula (persona natural) NO debe quedar marcado como "1" (Persona Jurídica) solo porque
// ese era el default fijo de antes — debe derivarse "2" del tipo de identificación.
func TestRegisterIssuer_NaturalPerson_EntityTypeCode(t *testing.T) {
	tests := []struct {
		idType     string
		wantEntity string
	}{
		{"13", "2"}, // Cédula de Ciudadanía -> Persona Natural
		{"11", "2"}, // Registro Civil
		{"22", "2"}, // Cédula de Extranjería
		{"41", "2"}, // Pasaporte
		{"31", "1"}, // NIT -> Persona Jurídica
		{"47", "1"}, // NIT de otro país
		{"50", "1"}, // NIT de la DIAN
	}
	for _, tt := range tests {
		t.Run(tt.idType, func(t *testing.T) {
			iss := validIssuer()
			iss.IdentificationTypeCode = tt.idType
			got, err := newService().RegisterIssuer(context.Background(), iss)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEntity, got.EntityTypeCode)
		})
	}
}

// TestRegisterIssuer_DerivesCheckDigit confirma el punto A de la sección 9.41: el dígito de
// verificación de un NIT ("31") se deriva con el algoritmo módulo 11 (internal/nit), nunca se
// acepta del cliente — el fixture manda "1" a propósito (un valor cualquiera, nunca validado
// antes de este cambio) y debe quedar sobreescrito con el dígito real ("4", verificado a mano).
func TestRegisterIssuer_DerivesCheckDigit(t *testing.T) {
	svc := newService()
	got, err := svc.RegisterIssuer(context.Background(), validIssuer())
	require.NoError(t, err)
	assert.Equal(t, "4", got.CheckDigit)
}

// TestRegisterIssuer_CheckDigitNotDerivedForNonNIT confirma que el concepto no aplica a otros
// tipos de identificación (cédula, etc.) — lo que el cliente mande ahí se conserva tal cual.
func TestRegisterIssuer_CheckDigitNotDerivedForNonNIT(t *testing.T) {
	iss := validIssuer()
	iss.IdentificationTypeCode = "13"
	iss.NIT = "6382356"
	iss.CheckDigit = "valor-cualquiera"

	got, err := newService().RegisterIssuer(context.Background(), iss)
	require.NoError(t, err)
	assert.Equal(t, "valor-cualquiera", got.CheckDigit)
}

// TestRegisterIssuer_InvalidNITForCheckDigit confirma el rechazo cuando el NIT no es numérico
// — no se puede derivar un dígito de verificación de algo que no es un número.
func TestRegisterIssuer_InvalidNITForCheckDigit(t *testing.T) {
	iss := validIssuer()
	iss.NIT = "90037-ABC"

	_, err := newService().RegisterIssuer(context.Background(), iss)
	assert.ErrorIs(t, err, issuers.ErrInvalidIdentificationNumber)
}

func TestRegisterIssuer_DuplicateNIT(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	_, err := svc.RegisterIssuer(ctx, validIssuer())
	require.NoError(t, err)

	_, err = svc.RegisterIssuer(ctx, validIssuer())
	assert.ErrorIs(t, err, issuers.ErrNITAlreadyExists)
}

func TestRegisterIssuer_Validations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*issuers.Issuer)
		wantErr error
	}{
		{"sin NIT", func(i *issuers.Issuer) { i.NIT = "" }, issuers.ErrEmptyNIT},
		{"sin razón social", func(i *issuers.Issuer) { i.BusinessName = "" }, issuers.ErrEmptyBusinessName},
		{"ambiente inválido", func(i *issuers.Issuer) { i.Environment = "9" }, issuers.ErrInvalidEnvironment},
		{"más de 4 códigos CIIU", func(i *issuers.Issuer) {
			i.IndustryClassificationCodes = []string{"4661", "4669", "4690", "4711", "4719"}
		}, issuers.ErrTooManyIndustryClassificationCodes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iss := validIssuer()
			tt.mutate(&iss)

			_, err := newService().RegisterIssuer(context.Background(), iss)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestRegisterIssuer_FourIndustryClassificationCodes_OK confirma el límite exacto (1
// actividad principal + 3 secundarias = 4, ver ErrTooManyIndustryClassificationCodes): 4
// códigos se aceptan, 5 se rechazan (ya cubierto en TestRegisterIssuer_Validations).
func TestRegisterIssuer_FourIndustryClassificationCodes_OK(t *testing.T) {
	iss := validIssuer()
	iss.IndustryClassificationCodes = []string{"4661", "4669", "4690", "4711"}
	got, err := newService().RegisterIssuer(context.Background(), iss)
	require.NoError(t, err)
	assert.Equal(t, iss.IndustryClassificationCodes, got.IndustryClassificationCodes)
}

// TestRegisterIssuer_InvalidLiabilityCode confirma el hallazgo de la auditoría de catálogos
// huérfanos: liability_codes es TEXT[], sin FK posible contra cada elemento, así que un
// código que no existe en el catálogo se rechaza aquí — antes de este fix, no se rechazaba
// en ningún lado hasta que la DIAN lo hiciera al confirmar un documento.
func TestRegisterIssuer_InvalidLiabilityCode(t *testing.T) {
	svc := issuers.New(issuers.NewMemoryRepository(), func([]byte, string) error { return nil }, nil, &fakeCatalogPort{valid: false})
	iss := validIssuer()
	iss.LiabilityCodes = []string{"CODIGO-INVENTADO"}

	_, err := svc.RegisterIssuer(context.Background(), iss)
	assert.ErrorIs(t, err, issuers.ErrInvalidLiabilityCode)
}

// TestRegisterIssuer_WithoutCredentials_OK confirma el cambio central de esta fase: el
// registro inicial ya NO exige software/PIN/certificado — solo los datos que la DIAN pide del
// emisor mismo. Se completan después, independientemente, vía UpdateIssuer (ver
// docs/apidian-architecture.md sección 9.25).
func TestRegisterIssuer_WithoutCredentials_OK(t *testing.T) {
	iss := validIssuer()
	iss.SoftwareID, iss.SoftwarePIN, iss.Certificate, iss.CertificatePassword = "", "", nil, ""

	got, err := newService().RegisterIssuer(context.Background(), iss)
	require.NoError(t, err)
	assert.Empty(t, got.SoftwareID)
	assert.Empty(t, got.Certificate)
}

func TestUpdateIssuer_OK(t *testing.T) {
	svc := newService()
	iss := validIssuer()
	iss.SoftwareID, iss.SoftwarePIN, iss.Certificate, iss.CertificatePassword = "", "", nil, ""
	created, err := svc.RegisterIssuer(context.Background(), iss)
	require.NoError(t, err)

	softwareID := "software-id-nuevo"
	pin := "54321"
	pwd := "clave-certificado-nueva"
	updated, err := svc.UpdateIssuer(context.Background(), created.ID, issuers.UpdateIssuerRequest{
		SoftwareID:          &softwareID,
		SoftwarePIN:         &pin,
		Certificate:         []byte("contenido-p12-nuevo"),
		CertificatePassword: &pwd,
	})
	require.NoError(t, err)
	assert.Equal(t, softwareID, updated.SoftwareID)
	assert.Equal(t, pin, updated.SoftwarePIN)
	assert.Equal(t, []byte("contenido-p12-nuevo"), updated.Certificate)
	assert.Equal(t, pwd, updated.CertificatePassword)
}

// TestUpdateIssuer_PartialUpdate confirma el requisito explícito del usuario: completar
// software/resolución/certificado INDEPENDIENTEMENTE, en el orden en que se vayan
// consiguiendo — actualizar solo SoftwareID no debe tocar lo que ya se había cargado antes.
func TestUpdateIssuer_PartialUpdate(t *testing.T) {
	svc := newService()
	created, err := svc.RegisterIssuer(context.Background(), validIssuer())
	require.NoError(t, err)

	newPIN := "99999"
	updated, err := svc.UpdateIssuer(context.Background(), created.ID, issuers.UpdateIssuerRequest{
		SoftwarePIN: &newPIN,
	})
	require.NoError(t, err)
	assert.Equal(t, newPIN, updated.SoftwarePIN)
	assert.Equal(t, created.SoftwareID, updated.SoftwareID, "no se tocó, debe seguir igual")
	assert.Equal(t, created.Certificate, updated.Certificate, "no se tocó, debe seguir igual")
}

func TestUpdateIssuer_EmptyValueRejected(t *testing.T) {
	svc := newService()
	created, err := svc.RegisterIssuer(context.Background(), validIssuer())
	require.NoError(t, err)

	empty := ""
	tests := []struct {
		name    string
		req     issuers.UpdateIssuerRequest
		wantErr error
	}{
		{"software_id vacío", issuers.UpdateIssuerRequest{SoftwareID: &empty}, issuers.ErrEmptySoftwareID},
		{"software_pin vacío", issuers.UpdateIssuerRequest{SoftwarePIN: &empty}, issuers.ErrEmptySoftwarePIN},
		{"certificado vacío", issuers.UpdateIssuerRequest{Certificate: []byte{}}, issuers.ErrEmptyCertificate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateIssuer(context.Background(), created.ID, tt.req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestUpdateIssuer_NotFound(t *testing.T) {
	svc := newService()
	softwareID := "x"
	_, err := svc.UpdateIssuer(context.Background(), uuid.New(), issuers.UpdateIssuerRequest{SoftwareID: &softwareID})
	assert.ErrorIs(t, err, issuers.ErrIssuerNotFound)
}

// TestUpdateIssuer_InvalidCertificate confirma el hallazgo de la sección 9.26 del
// architecture doc: subir un .p12 corrupto o con la contraseña equivocada debe fallar AQUÍ,
// con un error de dominio claro, no recién al confirmar un documento.
func TestUpdateIssuer_InvalidCertificate(t *testing.T) {
	rejecting := func([]byte, string) error { return errors.New("contraseña incorrecta") }
	svc := issuers.New(issuers.NewMemoryRepository(), rejecting, nil, &fakeCatalogPort{valid: true})
	created, err := svc.RegisterIssuer(context.Background(), validIssuer())
	require.NoError(t, err)

	pwd := "clave-nueva"
	_, err = svc.UpdateIssuer(context.Background(), created.ID, issuers.UpdateIssuerRequest{
		Certificate:         []byte("p12-corrupto"),
		CertificatePassword: &pwd,
	})
	assert.ErrorIs(t, err, issuers.ErrInvalidCertificate)
}

// TestUpdateIssuer_CertificateWithoutPasswordSkipsValidation confirma que la validación NO
// se dispara cuando la combinación certificado+contraseña todavía está incompleta — el
// usuario puede subir el certificado hoy y la contraseña la próxima semana (configuración
// gradual, sección 9.25) sin que se le rechace una combinación que nunca pretendió completar
// todavía. El validador "explota" si se llama, para probar que de verdad nunca se invoca.
func TestUpdateIssuer_CertificateWithoutPasswordSkipsValidation(t *testing.T) {
	exploding := func([]byte, string) error { t.Fatal("el validador no debería llamarse todavía"); return nil }
	svc := issuers.New(issuers.NewMemoryRepository(), exploding, nil, &fakeCatalogPort{valid: true})
	iss := validIssuer()
	iss.SoftwareID, iss.SoftwarePIN, iss.Certificate, iss.CertificatePassword = "", "", nil, ""
	created, err := svc.RegisterIssuer(context.Background(), iss)
	require.NoError(t, err)

	updated, err := svc.UpdateIssuer(context.Background(), created.ID, issuers.UpdateIssuerRequest{
		Certificate: []byte("p12-de-prueba"),
	})
	require.NoError(t, err)
	assert.Equal(t, []byte("p12-de-prueba"), updated.Certificate)
	assert.Empty(t, updated.CertificatePassword, "todavía no se completó, sigue vacía")
}

// TestDeleteLogo_OK confirma el punto #15: UpdateIssuer no puede expresar "bórralo" (Logo nil
// ya significa "no tocar"), así que DeleteLogo es la operación dedicada que limpia ambos
// campos.
func TestDeleteLogo_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	created, err := svc.RegisterIssuer(ctx, validIssuer())
	require.NoError(t, err)

	logo := []byte("contenido-png-de-prueba")
	contentType := "png"
	updated, err := svc.UpdateIssuer(ctx, created.ID, issuers.UpdateIssuerRequest{
		Logo:            logo,
		LogoContentType: &contentType,
	})
	require.NoError(t, err)
	require.NotEmpty(t, updated.Logo)

	got, err := svc.DeleteLogo(ctx, created.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Logo)
	assert.Empty(t, got.LogoContentType)
}

func TestDeleteLogo_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.DeleteLogo(context.Background(), uuid.New())
	assert.ErrorIs(t, err, issuers.ErrIssuerNotFound)
}

func TestGetIssuer_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.GetIssuer(context.Background(), uuid.New())
	assert.ErrorIs(t, err, issuers.ErrIssuerNotFound)
}

func TestGetIssuerByNIT_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	created, err := svc.RegisterIssuer(ctx, validIssuer())
	require.NoError(t, err)

	found, err := svc.GetIssuerByNIT(ctx, created.NIT)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
}
