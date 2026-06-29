package products_test

import (
	"context"
	"testing"

	"github.com/diegofxm/apidian/internal/products"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCatalogPort replica el subconjunto de tax_types que usan estos tests.
type fakeCatalogPort struct{}

func (fakeCatalogPort) GetTaxTypeName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"ZZ": "No aplica", "01": "IVA"}
	name, ok := names[code]
	return name, ok, nil
}

// GetItemStandardName/GetItemStandardAgencyID replican la tabla 13.3.5 real (sección 9.45) —
// solo "999" (estándar propio) tiene AgencyID vacío, a propósito.
func (fakeCatalogPort) GetItemStandardName(_ context.Context, code string) (string, bool, error) {
	names := map[string]string{"001": "UNSPSC", "010": "GTIN", "020": "Partida Arancelaria", "999": "Estándar de adopción del contribuyente"}
	name, ok := names[code]
	return name, ok, nil
}

func (fakeCatalogPort) GetItemStandardAgencyID(_ context.Context, code string) (string, bool, error) {
	agencyIDs := map[string]string{"001": "10", "010": "9", "020": "195", "999": ""}
	agencyID, ok := agencyIDs[code]
	return agencyID, ok, nil
}

func newService() *products.Service {
	return products.New(products.NewMemoryRepository(), fakeCatalogPort{})
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

// TestCreateProduct_DefaultsItemStandard confirma el fix de la sección 9.45: item_type_code
// vacío debe defaultear a "999" (estándar propio) y derivar item_type_name/
// item_type_agency_id del catálogo, nunca aceptarlos del cliente.
func TestCreateProduct_DefaultsItemStandard(t *testing.T) {
	svc := newService()
	p := validProduct()
	p.ItemTypeCode = ""
	p.ItemTypeName = ""

	got, err := svc.CreateProduct(context.Background(), uuid.New(), p)
	require.NoError(t, err)

	assert.Equal(t, "999", got.ItemTypeCode)
	assert.Equal(t, "Estándar de adopción del contribuyente", got.ItemTypeName)
	assert.Empty(t, got.ItemTypeAgencyID, "la fila 999 no debe traer agencyID")
}

// TestCreateProduct_DerivesItemStandardWithAgencyID confirma el otro lado: un código real
// ("001" UNSPSC) sí deriva un AgencyID no vacío.
func TestCreateProduct_DerivesItemStandardWithAgencyID(t *testing.T) {
	svc := newService()
	p := validProduct()
	p.ItemTypeCode = "001"

	got, err := svc.CreateProduct(context.Background(), uuid.New(), p)
	require.NoError(t, err)

	assert.Equal(t, "UNSPSC", got.ItemTypeName)
	assert.Equal(t, "10", got.ItemTypeAgencyID)
}

// TestCreateProduct_InvalidItemStandardCode confirma el rechazo cuando item_type_code no
// existe en la tabla 13.3.5 — exactamente el bug real (un código UNSPSC puntual puesto donde
// debía ir el selector).
func TestCreateProduct_InvalidItemStandardCode(t *testing.T) {
	svc := newService()
	p := validProduct()
	p.ItemTypeCode = "43211500"

	_, err := svc.CreateProduct(context.Background(), uuid.New(), p)
	assert.ErrorIs(t, err, products.ErrInvalidItemStandardCode)
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

	productA1 := validProduct()
	productA1.ItemCode = "SVC-001"
	_, err := svc.CreateProduct(ctx, issuerA, productA1)
	require.NoError(t, err)

	productA2 := validProduct()
	productA2.ItemCode = "SVC-002"
	_, err = svc.CreateProduct(ctx, issuerA, productA2)
	require.NoError(t, err)

	_, err = svc.CreateProduct(ctx, issuerB, validProduct())
	require.NoError(t, err)

	got, err := svc.ListProducts(ctx, issuerA)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

// TestCreateProduct_DuplicateItemCode confirma el fix del punto #3: dos productos del mismo
// emisor no pueden compartir item_code — antes no había ninguna validación al respecto.
func TestCreateProduct_DuplicateItemCode(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	_, err := svc.CreateProduct(ctx, issuerID, validProduct())
	require.NoError(t, err)

	_, err = svc.CreateProduct(ctx, issuerID, validProduct())
	assert.ErrorIs(t, err, products.ErrDuplicateItemCode)
}

// TestCreateProduct_SameItemCodeDifferentIssuer confirma que la unicidad es por emisor, no
// global — el mismo código de ítem en otro emisor no debe chocar.
func TestCreateProduct_SameItemCodeDifferentIssuer(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	_, err := svc.CreateProduct(ctx, issuerA, validProduct())
	require.NoError(t, err)

	_, err = svc.CreateProduct(ctx, issuerB, validProduct())
	assert.NoError(t, err)
}

// TestCreateProduct_EmptyItemCodeNeverDuplicates confirma que varios productos sin item_code
// (caso común, es opcional) conviven sin chocar entre sí.
func TestCreateProduct_EmptyItemCodeNeverDuplicates(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	p := validProduct()
	p.ItemCode = ""

	_, err := svc.CreateProduct(ctx, issuerID, p)
	require.NoError(t, err)
	_, err = svc.CreateProduct(ctx, issuerID, p)
	assert.NoError(t, err)
}

// TestUpdateProduct_DuplicateItemCode confirma que la validación también aplica al editar un
// producto existente contra el item_code de otro.
func TestUpdateProduct_DuplicateItemCode(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	first := validProduct()
	first.ItemCode = "SVC-001"
	_, err := svc.CreateProduct(ctx, issuerID, first)
	require.NoError(t, err)

	second := validProduct()
	second.ItemCode = "SVC-002"
	created, err := svc.CreateProduct(ctx, issuerID, second)
	require.NoError(t, err)

	updated := validProduct()
	updated.ItemCode = "SVC-001"
	_, err = svc.UpdateProduct(ctx, issuerID, created.ID, updated)
	assert.ErrorIs(t, err, products.ErrDuplicateItemCode)
}

// TestUpdateProduct_SameItemCodeAsItself confirma que actualizar un producto sin cambiar su
// propio item_code no se confunde con un duplicado contra sí mismo.
func TestUpdateProduct_SameItemCodeAsItself(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerID := uuid.New()

	created, err := svc.CreateProduct(ctx, issuerID, validProduct())
	require.NoError(t, err)

	updated := validProduct()
	updated.UnitPriceCents = 999999
	_, err = svc.UpdateProduct(ctx, issuerID, created.ID, updated)
	assert.NoError(t, err)
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
