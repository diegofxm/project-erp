package customers_test

import (
	"context"
	"testing"

	"github.com/diegofxm/apidian/internal/edocuments/customers"
	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCatalogPort acepta cualquier código por defecto — TestCreateCustomer_InvalidLiabilityCode
// pone valid en false explícitamente para probar el rechazo.
type fakeCatalogPort struct {
	valid bool
}

func (f *fakeCatalogPort) IsValidLiabilityCode(_ context.Context, _ string) (bool, error) {
	return f.valid, nil
}

// GetTaxTypeName replica el subconjunto de tax_types que usan estos tests.
func (f *fakeCatalogPort) GetTaxTypeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"ZZ": "No aplica", "01": "IVA"}
	name, ok := names[code]
	return name, ok, nil
}

func newService() *customers.Service {
	return customers.New(customers.NewMemoryRepository(), &fakeCatalogPort{valid: true})
}

func validParty() domain.Party {
	return domain.Party{
		Identification: domain.Identification{Number: "222222222222", TypeCode: "13"},
		Name:           "Consumidor Final",
		Address:        domain.Address{Line: "Calle 1", CityCode: "11001", CityName: "Bogotá"},
		Email:          "cliente@empresa.test",
	}
}

func TestCreateCustomer_OK(t *testing.T) {
	svc := newService()
	issuerID := uuid.New()

	c, err := svc.CreateCustomer(context.Background(), issuerID, validParty())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, issuerID, c.IssuerID)
	assert.Equal(t, "Consumidor Final", c.Party.Name)
	assert.False(t, c.CreatedAt.IsZero())
}

func TestCreateCustomer_Validations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*domain.Party)
		wantErr error
	}{
		{"sin nombre", func(p *domain.Party) { p.Name = "" }, customers.ErrEmptyName},
		{"sin identificación", func(p *domain.Party) { p.Identification.Number = "" }, customers.ErrEmptyIdentification},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validParty()
			tt.mutate(&p)
			_, err := newService().CreateCustomer(context.Background(), uuid.New(), p)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestCreateCustomer_InvalidLiabilityCode confirma el hallazgo de la auditoría de catálogos
// huérfanos: liability_codes es TEXT[], sin FK posible contra cada elemento, así que un
// código que no existe en el catálogo se rechaza aquí.
func TestCreateCustomer_InvalidLiabilityCode(t *testing.T) {
	svc := customers.New(customers.NewMemoryRepository(), &fakeCatalogPort{valid: false})
	p := validParty()
	p.LiabilityCodes = []string{"CODIGO-INVENTADO"}

	_, err := svc.CreateCustomer(context.Background(), uuid.New(), p)
	assert.ErrorIs(t, err, customers.ErrInvalidLiabilityCode)
}

// TestCreateCustomer_DerivesVerificationCodeForNIT confirma el punto A de la sección 9.41: un
// NIT ("31") deriva su dígito de verificación con el algoritmo módulo 11 (internal/nit) — el
// número 6382356 es un caso real verificado (dígito 7), no inventado.
func TestCreateCustomer_DerivesVerificationCodeForNIT(t *testing.T) {
	p := validParty()
	p.Identification = domain.Identification{Number: "6382356", TypeCode: "31", VerificationCode: "9"} // "9" a propósito, debe sobreescribirse

	got, err := newService().CreateCustomer(context.Background(), uuid.New(), p)
	require.NoError(t, err)
	assert.Equal(t, "7", got.Party.Identification.VerificationCode)
}

// TestCreateCustomer_VerificationCodeNotDerivedForNonNIT confirma que el concepto no aplica a
// otros tipos de identificación — lo que el cliente mande ahí (o nada) se conserva tal cual.
func TestCreateCustomer_VerificationCodeNotDerivedForNonNIT(t *testing.T) {
	got, err := newService().CreateCustomer(context.Background(), uuid.New(), validParty())
	require.NoError(t, err)
	assert.Empty(t, got.Party.Identification.VerificationCode)
}

// TestCreateCustomer_InvalidNITForVerificationCode confirma el rechazo cuando el número no es
// numérico — no se puede derivar un dígito de verificación de algo que no es un NIT real.
func TestCreateCustomer_InvalidNITForVerificationCode(t *testing.T) {
	p := validParty()
	p.Identification = domain.Identification{Number: "ABC123", TypeCode: "31"}

	_, err := newService().CreateCustomer(context.Background(), uuid.New(), p)
	assert.ErrorIs(t, err, customers.ErrInvalidIdentificationNumber)
}

func TestGetCustomer_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.GetCustomer(context.Background(), uuid.New())
	assert.ErrorIs(t, err, customers.ErrCustomerNotFound)
}

func TestListCustomers_OnlyOwnIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	_, err := svc.CreateCustomer(ctx, issuerA, validParty())
	require.NoError(t, err)
	_, err = svc.CreateCustomer(ctx, issuerA, validParty())
	require.NoError(t, err)
	_, err = svc.CreateCustomer(ctx, issuerB, validParty())
	require.NoError(t, err)

	got, err := svc.ListCustomers(ctx, issuerA)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestUpdateCustomer_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	created, err := svc.CreateCustomer(ctx, issuerID, validParty())
	require.NoError(t, err)

	updated := validParty()
	updated.Name = "Nombre Actualizado"
	got, err := svc.UpdateCustomer(ctx, issuerID, created.ID, updated)
	require.NoError(t, err)
	assert.Equal(t, "Nombre Actualizado", got.Party.Name)
}

func TestUpdateCustomer_OtherIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	created, err := svc.CreateCustomer(ctx, issuerA, validParty())
	require.NoError(t, err)

	_, err = svc.UpdateCustomer(ctx, issuerB, created.ID, validParty())
	assert.ErrorIs(t, err, customers.ErrCustomerNotFound)
}

func TestDeleteCustomer_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	created, err := svc.CreateCustomer(ctx, issuerID, validParty())
	require.NoError(t, err)

	require.NoError(t, svc.DeleteCustomer(ctx, issuerID, created.ID))
	_, err = svc.GetCustomer(ctx, created.ID)
	assert.ErrorIs(t, err, customers.ErrCustomerNotFound)
}

func TestDeleteCustomer_OtherIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	created, err := svc.CreateCustomer(ctx, issuerA, validParty())
	require.NoError(t, err)

	err = svc.DeleteCustomer(ctx, issuerB, created.ID)
	assert.ErrorIs(t, err, customers.ErrCustomerNotFound)
}
