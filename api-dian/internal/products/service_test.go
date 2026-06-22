package products_test

import (
	"context"
	"testing"

	"github.com/diegofxm/api-dian/internal/products"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *products.Service {
	return products.New(products.NewMemoryRepository())
}

func validProduct() products.Product {
	return products.Product{
		Description:    "Servicio de consultoría",
		UnitCode:       "94",
		UnitPriceCents: 100000,
		ItemCode:       "SVC-001",
		ItemTypeCode:   "999",
		ItemTypeName:   "Estándar de adopción del contribuyente",
		TaxTypeCode:    "01",
		TaxTypeName:    "IVA",
		TaxPercent:     19,
	}
}

func TestCreateProduct_OK(t *testing.T) {
	svc := newService()
	issuerID := uuid.New()

	p, err := svc.CreateProduct(context.Background(), issuerID, validProduct())
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, issuerID, p.IssuerID)
	assert.Equal(t, "Servicio de consultoría", p.Description)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestCreateProduct_Validations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*products.Product)
		wantErr error
	}{
		{"sin descripción", func(p *products.Product) { p.Description = "" }, products.ErrEmptyDescription},
		{"sin unidad de medida", func(p *products.Product) { p.UnitCode = "" }, products.ErrEmptyUnitCode},
		{"precio negativo", func(p *products.Product) { p.UnitPriceCents = -1 }, products.ErrInvalidUnitPrice},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProduct()
			tt.mutate(&p)
			_, err := newService().CreateProduct(context.Background(), uuid.New(), p)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.GetProduct(context.Background(), uuid.New())
	assert.ErrorIs(t, err, products.ErrProductNotFound)
}

func TestListProducts_OnlyOwnIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	_, err := svc.CreateProduct(ctx, issuerA, validProduct())
	require.NoError(t, err)
	_, err = svc.CreateProduct(ctx, issuerA, validProduct())
	require.NoError(t, err)
	_, err = svc.CreateProduct(ctx, issuerB, validProduct())
	require.NoError(t, err)

	got, err := svc.ListProducts(ctx, issuerA)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestUpdateProduct_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	created, err := svc.CreateProduct(ctx, issuerID, validProduct())
	require.NoError(t, err)

	updated := validProduct()
	updated.UnitPriceCents = 200000
	got, err := svc.UpdateProduct(ctx, issuerID, created.ID, updated)
	require.NoError(t, err)
	assert.Equal(t, int64(200000), got.UnitPriceCents)
}

func TestUpdateProduct_OtherIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	created, err := svc.CreateProduct(ctx, issuerA, validProduct())
	require.NoError(t, err)

	_, err = svc.UpdateProduct(ctx, issuerB, created.ID, validProduct())
	assert.ErrorIs(t, err, products.ErrProductNotFound)
}

func TestDeleteProduct_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	created, err := svc.CreateProduct(ctx, issuerID, validProduct())
	require.NoError(t, err)

	require.NoError(t, svc.DeleteProduct(ctx, issuerID, created.ID))
	_, err = svc.GetProduct(ctx, created.ID)
	assert.ErrorIs(t, err, products.ErrProductNotFound)
}

func TestDeleteProduct_OtherIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	created, err := svc.CreateProduct(ctx, issuerA, validProduct())
	require.NoError(t, err)

	err = svc.DeleteProduct(ctx, issuerB, created.ID)
	assert.ErrorIs(t, err, products.ErrProductNotFound)
}
