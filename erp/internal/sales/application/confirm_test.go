package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	inventorydomain "github.com/diegofxm/erp/internal/inventory/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	"github.com/diegofxm/erp/internal/sales/application"
	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeSaleRepo struct {
	sale domain.Sale
}

func (f *fakeSaleRepo) Save(context.Context, domain.Sale) (*domain.Sale, error) { return &f.sale, nil }
func (f *fakeSaleRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Sale, error) {
	return &f.sale, nil
}
func (f *fakeSaleRepo) List(context.Context, uuid.UUID) ([]domain.Sale, error) { return nil, nil }
func (f *fakeSaleRepo) UpdateStatus(context.Context, uuid.UUID, uuid.UUID, domain.SaleStatus) error {
	return nil
}
func (f *fakeSaleRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeSaleRepo) SetInvoiceDocumentID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return nil
}
func (f *fakeSaleRepo) NextSaleNumber(context.Context, uuid.UUID, int) (int, error) { return 1, nil }
func (f *fakeSaleRepo) SetSaleNumberCounter(context.Context, uuid.UUID, int, int) (int, error) {
	return 1, nil
}

type fakeProductRepo struct {
	products map[uuid.UUID]productdomain.Product
}

func (f *fakeProductRepo) Save(context.Context, productdomain.Product) (*productdomain.Product, error) {
	return nil, nil
}
func (f *fakeProductRepo) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*productdomain.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return nil, productdomain.ErrProductNotFound
	}
	return &p, nil
}
func (f *fakeProductRepo) GetByCode(context.Context, uuid.UUID, string) (*productdomain.Product, error) {
	return nil, nil
}
func (f *fakeProductRepo) List(context.Context, uuid.UUID) ([]productdomain.Product, error) {
	return nil, nil
}
func (f *fakeProductRepo) Update(context.Context, productdomain.Product) (*productdomain.Product, error) {
	return nil, nil
}
func (f *fakeProductRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type fakeInventoryRepo struct {
	stock map[uuid.UUID]float64 // por product_id
}

func (f *fakeInventoryRepo) GetStock(_ context.Context, _ uuid.UUID, productID, _ uuid.UUID) (*inventorydomain.StockEntry, error) {
	return &inventorydomain.StockEntry{ProductID: productID, Quantity: f.stock[productID]}, nil
}
func (f *fakeInventoryRepo) ListStock(context.Context, uuid.UUID) ([]inventorydomain.StockEntry, error) {
	return nil, nil
}
func (f *fakeInventoryRepo) UpsertStock(context.Context, inventorydomain.StockEntry) error {
	return nil
}
func (f *fakeInventoryRepo) SaveMovement(_ context.Context, m inventorydomain.Movement) (*inventorydomain.Movement, error) {
	return &m, nil
}
func (f *fakeInventoryRepo) ListMovements(context.Context, uuid.UUID, *uuid.UUID) ([]inventorydomain.Movement, error) {
	return nil, nil
}
func (f *fakeInventoryRepo) Transfer(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, float64, string, string) (*inventorydomain.Movement, *inventorydomain.Movement, error) {
	return nil, nil, nil
}

type fakeCustomerPort struct {
	customer domain.Customer
}

func (f *fakeCustomerPort) GetByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Customer, error) {
	return &f.customer, nil
}

type fakePaymentRepo struct {
	receivables []domain.ReceivableBalance
}

func (f *fakePaymentRepo) Save(context.Context, domain.SalePayment) (*domain.SalePayment, error) {
	return nil, nil
}
func (f *fakePaymentRepo) ListBySale(context.Context, uuid.UUID, uuid.UUID) ([]domain.SalePayment, error) {
	return nil, nil
}
func (f *fakePaymentRepo) GetReceivables(context.Context, uuid.UUID) ([]domain.ReceivableBalance, error) {
	return f.receivables, nil
}
func (f *fakePaymentRepo) GetReceivablesByCustomer(context.Context, uuid.UUID, uuid.UUID) ([]domain.ReceivableBalance, error) {
	return f.receivables, nil
}

type fakeWarehouseRepo struct {
	warehouse companydomain.Warehouse
}

func (f *fakeWarehouseRepo) Save(context.Context, companydomain.Warehouse) (*companydomain.Warehouse, error) {
	return nil, nil
}
func (f *fakeWarehouseRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*companydomain.Warehouse, error) {
	return &f.warehouse, nil
}
func (f *fakeWarehouseRepo) List(context.Context, uuid.UUID) ([]companydomain.Warehouse, error) {
	return nil, nil
}
func (f *fakeWarehouseRepo) Update(context.Context, companydomain.Warehouse) (*companydomain.Warehouse, error) {
	return nil, nil
}
func (f *fakeWarehouseRepo) Deactivate(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeWarehouseRepo) SetDefault(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeWarehouseRepo) GetOrCreateDefault(context.Context, uuid.UUID) (*companydomain.Warehouse, error) {
	return &f.warehouse, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newConfirmTestUseCase(t *testing.T) (*application.ConfirmUseCase, *fakeSaleRepo, *fakeInventoryRepo, *fakeCustomerPort, *fakePaymentRepo, *fakeProductRepo) {
	t.Helper()
	companyID := uuid.New()
	productID := uuid.New()
	customerID := uuid.New()
	warehouseID := uuid.New()

	saleRepo := &fakeSaleRepo{sale: domain.Sale{
		ID: uuid.New(), CompanyID: companyID, CustomerID: customerID, Status: domain.StatusDraft,
		Lines: []domain.SaleLine{{ProductID: productID, Quantity: 2, UnitPrice: 50000, Total: 100000}},
	}}
	productRepo := &fakeProductRepo{products: map[uuid.UUID]productdomain.Product{
		productID: {ID: productID, Name: "Producto Test", IsService: false},
	}}
	inventoryRepo := &fakeInventoryRepo{stock: map[uuid.UUID]float64{productID: 10}}
	customerPort := &fakeCustomerPort{customer: domain.Customer{Name: "Cliente Test"}}
	paymentRepo := &fakePaymentRepo{}
	warehouseRepo := &fakeWarehouseRepo{warehouse: companydomain.Warehouse{ID: warehouseID, CompanyID: companyID, IsDefault: true}}

	bus := events.NewBus()
	uc := application.NewConfirmUseCase(saleRepo, bus, productRepo, inventoryRepo, customerPort, paymentRepo, warehouseRepo, nil)
	return uc, saleRepo, inventoryRepo, customerPort, paymentRepo, productRepo
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestConfirm_RejectsInsufficientStock(t *testing.T) {
	uc, saleRepo, inventoryRepo, _, _, _ := newConfirmTestUseCase(t)
	inventoryRepo.stock[saleRepo.sale.Lines[0].ProductID] = 1 // pidió 2, solo hay 1

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("esperaba ErrInsufficientStock, got %v", err)
	}
}

func TestConfirm_NoCreditLimit_AllowsAnyAmount(t *testing.T) {
	uc, saleRepo, _, customerPort, _, _ := newConfirmTestUseCase(t)
	customerPort.customer.CreditLimit = nil // sin cupo = sin límite

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if err != nil {
		t.Fatalf("no esperaba error sin cupo configurado: %v", err)
	}
}

func TestConfirm_WithinCreditLimit_Allowed(t *testing.T) {
	uc, saleRepo, _, customerPort, _, _ := newConfirmTestUseCase(t)
	limit := 1000000.0
	customerPort.customer.CreditLimit = &limit // venta es 100000, muy por debajo

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if err != nil {
		t.Fatalf("no esperaba error dentro del cupo: %v", err)
	}
}

func TestConfirm_ExceedsCreditLimit_Rejected(t *testing.T) {
	uc, saleRepo, _, customerPort, _, _ := newConfirmTestUseCase(t)
	limit := 50000.0 // venta es 100000, se pasa
	customerPort.customer.CreditLimit = &limit

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if !errors.Is(err, domain.ErrCreditLimitExceeded) {
		t.Fatalf("esperaba ErrCreditLimitExceeded, got %v", err)
	}
}

func TestConfirm_OverdueBalance_RejectedRegardlessOfLimit(t *testing.T) {
	uc, saleRepo, _, customerPort, paymentRepo, _ := newConfirmTestUseCase(t)
	customerPort.customer.CreditLimit = nil // sin límite configurado...
	pastDue := time.Now().Add(-48 * time.Hour)
	paymentRepo.receivables = []domain.ReceivableBalance{
		{SaleNumber: "V-1", DueDate: &pastDue, Balance: 1}, // ...pero hay cartera vencida
	}

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if !errors.Is(err, domain.ErrOverdueBalance) {
		t.Fatalf("esperaba ErrOverdueBalance, got %v", err)
	}
}

func TestConfirm_ServiceProduct_SkipsStockCheck(t *testing.T) {
	uc, saleRepo, inventoryRepo, _, _, productRepo := newConfirmTestUseCase(t)
	productID := saleRepo.sale.Lines[0].ProductID
	inventoryRepo.stock[productID] = 0 // sin stock disponible...

	// ...pero el producto es un servicio, así que no debería exigir stock.
	p := productRepo.products[productID]
	p.IsService = true
	productRepo.products[productID] = p

	_, err := uc.Execute(context.Background(), saleRepo.sale.CompanyID, saleRepo.sale.ID)
	if err != nil {
		t.Fatalf("no esperaba error de stock para un producto de servicio: %v", err)
	}
}
