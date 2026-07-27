package auth_test

import (
	"context"
	"testing"

	"github.com/diegofxm/apidian/internal/auth"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeIssuerPort es un doble mínimo — auth.Service no necesita un *issuers.Service real para
// probarse, solo lo que declara auth.IssuerPort (mismo patrón que documents.fakeIssuerPort).
// Guarda lo creado en un mapa para que GetIssuer pueda devolverlo de vuelta — necesario desde
// la Fase 9.32, donde Login/CreateIssuerForUser/SelectIssuer consultan el emisor por ID para
// devolverlo como "empresa activa".
type fakeIssuerPort struct {
	calls int
	err   error
	byID  map[uuid.UUID]*issuers.Issuer
}

func (f *fakeIssuerPort) RegisterIssuer(_ context.Context, iss issuers.Issuer) (*issuers.Issuer, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	iss.ID = uuid.New()
	if f.byID == nil {
		f.byID = make(map[uuid.UUID]*issuers.Issuer)
	}
	f.byID[iss.ID] = &iss
	return &iss, nil
}

func (f *fakeIssuerPort) GetIssuer(_ context.Context, id uuid.UUID) (*issuers.Issuer, error) {
	iss, ok := f.byID[id]
	if !ok {
		return nil, issuers.ErrIssuerNotFound
	}
	return iss, nil
}

func testTokens() *auth.TokenIssuer {
	return auth.NewTokenIssuer([]byte("clave-de-prueba-no-usar-en-produccion"))
}

func validRegisterRequest() auth.RegisterRequest {
	return auth.RegisterRequest{
		Email:    "admin@empresa.test",
		Password: "contraseña-segura",
		Name:     "Admin de Prueba",
	}
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
	}
}

func TestRegister_OK(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())

	result, err := svc.Register(context.Background(), validRegisterRequest())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.User.ID)
	assert.Equal(t, "admin@empresa.test", result.User.Email)
	assert.Equal(t, auth.RoleAdmin, result.User.Role)
	assert.True(t, result.User.IsActive)
	assert.NotEmpty(t, result.Token)
	assert.Nil(t, result.ActiveIssuer, "un usuario recién registrado no tiene ninguna empresa todavía")
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

// TestLogin_AutoSelectsSingleIssuer confirma el comportamiento elegido para el caso normal de
// hoy (un usuario con una sola empresa): el login la autoselecciona, sin un paso extra — ver
// docs/apidian-architecture.md sección 9.32.
func TestLogin_AutoSelectsSingleIssuer(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	registered, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)
	created, err := svc.CreateIssuerForUser(ctx, registered.User.ID, validIssuer())
	require.NoError(t, err)
	require.NotNil(t, created.ActiveIssuer)

	result, err := svc.Login(ctx, "admin@empresa.test", "contraseña-segura")
	require.NoError(t, err)
	require.NotNil(t, result.ActiveIssuer, "con una sola empresa vinculada, el login debe autoseleccionarla")
	assert.Equal(t, created.ActiveIssuer.ID, result.ActiveIssuer.ID)
}

// TestLogin_NoAutoSelectWithZeroOrManyIssuers: sin empresas, o con varias, el login no decide
// por el usuario — el token queda sin empresa activa.
func TestLogin_NoAutoSelectWithZeroOrManyIssuers(t *testing.T) {
	ctx := context.Background()

	t.Run("cero empresas", func(t *testing.T) {
		svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
		_, err := svc.Register(ctx, validRegisterRequest())
		require.NoError(t, err)

		result, err := svc.Login(ctx, "admin@empresa.test", "contraseña-segura")
		require.NoError(t, err)
		assert.Nil(t, result.ActiveIssuer)
	})

	t.Run("dos empresas", func(t *testing.T) {
		svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
		registered, err := svc.Register(ctx, validRegisterRequest())
		require.NoError(t, err)
		_, err = svc.CreateIssuerForUser(ctx, registered.User.ID, validIssuer())
		require.NoError(t, err)
		other := validIssuer()
		other.NIT = "900111222"
		_, err = svc.CreateIssuerForUser(ctx, registered.User.ID, other)
		require.NoError(t, err)

		result, err := svc.Login(ctx, "admin@empresa.test", "contraseña-segura")
		require.NoError(t, err)
		assert.Nil(t, result.ActiveIssuer, "con varias empresas, el login no debe adivinar cuál usar")
	})
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

// TestCreateIssuerForUser_ActivatesImmediately confirma que la empresa recién creada queda
// activa en el token devuelto, sin un paso extra de selección.
func TestCreateIssuerForUser_ActivatesImmediately(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	registered, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)

	result, err := svc.CreateIssuerForUser(ctx, registered.User.ID, validIssuer())
	require.NoError(t, err)
	require.NotNil(t, result.ActiveIssuer)
	assert.Equal(t, "Empresa de Prueba S.A.S.", result.ActiveIssuer.BusinessName)
}

func TestListUserIssuers(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	registered, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)

	list, err := svc.ListUserIssuers(ctx, registered.User.ID)
	require.NoError(t, err)
	assert.Empty(t, list)

	_, err = svc.CreateIssuerForUser(ctx, registered.User.ID, validIssuer())
	require.NoError(t, err)

	list, err = svc.ListUserIssuers(ctx, registered.User.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestSelectIssuer_OK(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	registered, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)
	created, err := svc.CreateIssuerForUser(ctx, registered.User.ID, validIssuer())
	require.NoError(t, err)

	result, err := svc.SelectIssuer(ctx, registered.User.ID, created.ActiveIssuer.ID)
	require.NoError(t, err)
	require.NotNil(t, result.ActiveIssuer)
	assert.Equal(t, created.ActiveIssuer.ID, result.ActiveIssuer.ID)
}

// TestSelectIssuer_AccessDenied confirma que un usuario no puede activar una empresa a la que
// nunca se vinculó — mismo criterio de aislamiento entre tenants que el resto de la API.
func TestSelectIssuer_AccessDenied(t *testing.T) {
	svc := auth.New(auth.NewMemoryRepository(), &fakeIssuerPort{}, testTokens())
	ctx := context.Background()

	userA, err := svc.Register(ctx, validRegisterRequest())
	require.NoError(t, err)
	otherReq := validRegisterRequest()
	otherReq.Email = "otro@empresa.test"
	userB, err := svc.Register(ctx, otherReq)
	require.NoError(t, err)
	issuerOfB, err := svc.CreateIssuerForUser(ctx, userB.User.ID, validIssuer())
	require.NoError(t, err)

	_, err = svc.SelectIssuer(ctx, userA.User.ID, issuerOfB.ActiveIssuer.ID)
	assert.ErrorIs(t, err, auth.ErrIssuerAccessDenied)
}

func TestTokenIssuer_VerifyRoundtrip(t *testing.T) {
	tokens := testTokens()
	u := auth.User{ID: uuid.New()}
	tenantID := uuid.New()

	token, err := tokens.Issue(u, tenantID)
	require.NoError(t, err)

	userID, gotTenantID, err := tokens.Verify(token)
	require.NoError(t, err)
	assert.Equal(t, u.ID, userID)
	assert.Equal(t, tenantID, gotTenantID)
}

func TestTokenIssuer_Issue_NoActiveTenant(t *testing.T) {
	u := auth.User{ID: uuid.New()}
	token, err := testTokens().Issue(u, uuid.Nil)
	require.NoError(t, err)

	_, tenantID, err := testTokens().Verify(token)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, tenantID)
}

func TestTokenIssuer_Verify_WrongSecret(t *testing.T) {
	u := auth.User{ID: uuid.New()}
	token, err := testTokens().Issue(u, uuid.New())
	require.NoError(t, err)

	other := auth.NewTokenIssuer([]byte("otra-clave-distinta"))
	_, _, err = other.Verify(token)
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}
