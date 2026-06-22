package auth_test

import (
	"context"
	"testing"

	"github.com/diegofxm/api-dian/internal/auth"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIssuerPort es un doble mínimo — auth.Service no necesita un *issuers.Service real para
// probarse, solo lo que declara auth.IssuerPort (mismo patrón que documents.fakeIssuerPort).
type fakeIssuerPort struct {
	calls int
	err   error
}

func (f *fakeIssuerPort) RegisterIssuer(_ context.Context, iss issuers.Issuer) (*issuers.Issuer, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	iss.ID = uuid.New()
	return &iss, nil
}

func testTokens() *auth.TokenIssuer {
	return auth.NewTokenIssuer([]byte("clave-de-prueba-no-usar-en-produccion"))
}

func validRegisterRequest() auth.RegisterRequest {
	return auth.RegisterRequest{
		Issuer: issuers.Issuer{
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
		},
		Email:    "admin@empresa.test",
		Password: "contraseña-segura",
		Name:     "Admin de Prueba",
	}
}

func TestRegister_OK(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())

	result, err := svc.Register(context.Background(), validRegisterRequest())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.User.ID)
	assert.NotEqual(t, uuid.Nil, result.User.IssuerID)
	assert.Equal(t, "admin@empresa.test", result.User.Email)
	assert.Equal(t, auth.RoleAdmin, result.User.Role)
	assert.True(t, result.User.IsActive)
	assert.NotEmpty(t, result.Token)
	assert.NotEqual(t, "contraseña-segura", result.User.PasswordHash, "la contraseña nunca debe quedar en texto plano")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	_, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)

	_, err = svc.Register(ctx, validRegisterRequest())
	assert.ErrorIs(t, err, auth.ErrEmailAlreadyExists)
}

func TestRegister_DuplicateEmail_NeverCreatesIssuer(t *testing.T) {
	port := &fakeIssuerPort{}
	svc := auth.New(auth.NewMemoryRepository(), port, testTokens())
	ctx := context.Background()

	_, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)
	require.Equal(t, 1, port.calls)

	// El segundo intento con el mismo correo debe fallar ANTES de llamar a RegisterIssuer —
	// si no, quedaría un emisor huérfano sin ningún usuario que lo administre.
	_, err = svc.Register(ctx, validRegisterRequest())
	require.ErrorIs(t, err, auth.ErrEmailAlreadyExists)
	assert.Equal(t, 1, port.calls, "RegisterIssuer no debe llamarse de nuevo si el correo ya existe")
}

func TestRegister_Validations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*auth.RegisterRequest)
		wantErr error
	}{
		{"sin correo", func(r *auth.RegisterRequest) { r.Email = "" }, auth.ErrEmptyEmail},
		{"sin contraseña", func(r *auth.RegisterRequest) { r.Password = "" }, auth.ErrEmptyPassword},
		{"contraseña corta", func(r *auth.RegisterRequest) { r.Password = "corta" }, auth.ErrPasswordTooShort},
		{"sin nombre", func(r *auth.RegisterRequest) { r.Name = "" }, auth.ErrEmptyName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validRegisterRequest()
			tt.mutate(&req)

			svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
			_, err := svc.Register(context.Background(), req)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLogin_OK(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	registered, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)

	result, err := svc.Login(ctx, "admin@empresa.test", "contraseña-segura")
	require.NoError(t, err)
	assert.Equal(t, registered.User.ID, result.User.ID)
	assert.NotEmpty(t, result.Token)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	_, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)

	_, err = svc.Login(ctx, "admin@empresa.test", "contraseña-equivocada")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())

	_, err := svc.Login(context.Background(), "no-existe@empresa.test", "lo-que-sea")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestTokenIssuer_VerifyRoundtrip(t *testing.T) {
	tokens := testTokens()
	u := auth.User{ID: uuid.New(), IssuerID: uuid.New()}

	token, err := tokens.Issue(u)
	require.NoError(t, err)

	userID, tenantID, err := tokens.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, u.ID, userID)
	assert.Equal(t, u.IssuerID, tenantID)
}

func TestTokenIssuer_Verify_WrongSecret(t *testing.T) {
	u := auth.User{ID: uuid.New(), IssuerID: uuid.New()}
	token, err := testTokens().Issue(u)
	require.NoError(t, err)

	other := auth.NewTokenIssuer([]byte("otra-clave-distinta"))
	_, _, err = other.Verify(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}
