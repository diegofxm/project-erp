package issuers_test

import (
	"context"
	"testing"

	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *issuers.Service {
	return issuers.New(issuers.NewMemoryRepository())
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
	assert.Equal(t, "1", iss.EntityTypeCode, "debería tomar el default validado contra la DIAN real")
	assert.Equal(t, "ZZ", iss.TaxSchemeCode)
	assert.Equal(t, "No aplica", iss.TaxSchemeName)
	assert.False(t, iss.CreatedAt.IsZero())
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
		{"sin software ID", func(i *issuers.Issuer) { i.SoftwareID = "" }, issuers.ErrEmptySoftwareID},
		{"sin PIN", func(i *issuers.Issuer) { i.SoftwarePIN = "" }, issuers.ErrEmptySoftwarePIN},
		{"sin certificado", func(i *issuers.Issuer) { i.Certificate = nil }, issuers.ErrEmptyCertificate},
		{"ambiente inválido", func(i *issuers.Issuer) { i.Environment = "9" }, issuers.ErrInvalidEnvironment},
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
