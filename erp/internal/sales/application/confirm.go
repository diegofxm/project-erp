package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	inventorydomain "github.com/diegofxm/erp/internal/inventory/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type ConfirmUseCase struct {
	repo       domain.Repository
	bus        *events.Bus
	products   productdomain.Repository
	inventory  inventorydomain.Repository
	customers  domain.CustomerPort
	payments   domain.PaymentRepository
	warehouses companydomain.WarehouseRepository
}

func NewConfirmUseCase(
	repo domain.Repository,
	bus *events.Bus,
	products productdomain.Repository,
	inventory inventorydomain.Repository,
	customers domain.CustomerPort,
	payments domain.PaymentRepository,
	warehouses companydomain.WarehouseRepository,
) *ConfirmUseCase {
	return &ConfirmUseCase{
		repo: repo, bus: bus, products: products, inventory: inventory,
		customers: customers, payments: payments, warehouses: warehouses,
	}
}

func (uc *ConfirmUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) (*domain.Sale, error) {
	s, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if s.Status != domain.StatusDraft {
		return nil, domain.ErrSaleNotDraft
	}

	warehouse, err := uc.warehouses.GetOrCreateDefault(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("resolver bodega por defecto: %w", err)
	}
	if err := uc.checkStock(ctx, companyID, warehouse.ID, s.Lines); err != nil {
		return nil, err
	}
	if err := uc.checkCredit(ctx, companyID, s); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusConfirmed); err != nil {
		return nil, err
	}
	s.Status = domain.StatusConfirmed

	_, tax, total := s.GrandTotal()
	uc.bus.Publish(domain.SaleConfirmed{
		SaleID:     s.ID,
		CompanyID:  s.CompanyID,
		CustomerID: s.CustomerID,
		Total:      total,
		TaxAmount:  tax,
		IssueDate:  s.IssueDate,
		Lines:      s.Lines,
	})

	return s, nil
}

// checkStock valida que haya suficiente inventario ANTES de confirmar. Antes de este cambio el
// descuento de stock ocurría de forma asíncrona vía evento (inventory/application/
// on_sale_confirmed.go) y si fallaba por falta de stock el error solo se registraba en el log
// del servidor — la venta quedaba "confirmada" igual, sin que el usuario se enterara de que
// vendió algo que no tenía. Los productos de servicio (IsService) no llevan inventario, se
// excluyen del chequeo.
func (uc *ConfirmUseCase) checkStock(ctx context.Context, companyID, warehouseID uuid.UUID, lines []domain.SaleLine) error {
	needed := map[uuid.UUID]float64{}
	for _, l := range lines {
		needed[l.ProductID] += l.Quantity
	}

	var shortfalls []string
	for productID, qty := range needed {
		p, err := uc.products.GetByID(ctx, companyID, productID)
		if err != nil || p.IsService {
			continue
		}
		stock, err := uc.inventory.GetStock(ctx, companyID, productID, warehouseID)
		var available float64
		if err == nil {
			available = stock.Quantity
		}
		if available < qty {
			shortfalls = append(shortfalls, fmt.Sprintf("%q (disponible %.2f, se necesitan %.2f)", p.Name, available, qty))
		}
	}
	if len(shortfalls) > 0 {
		return fmt.Errorf("%w: %s", domain.ErrInsufficientStock, strings.Join(shortfalls, "; "))
	}
	return nil
}

// checkCredit valida cartera vencida y cupo de crédito antes de confirmar. Antes de este cambio
// no existía ningún control: un cliente con facturas vencidas o muy por encima de su cupo podía
// seguir comprando a crédito sin restricción. Cartera vencida bloquea siempre, sin importar si
// hay cupo configurado; el cupo de crédito solo se valida si el cliente tiene uno (nil = sin
// límite).
func (uc *ConfirmUseCase) checkCredit(ctx context.Context, companyID uuid.UUID, s *domain.Sale) error {
	customer, err := uc.customers.GetByID(ctx, companyID, s.CustomerID)
	if err != nil {
		return nil
	}

	receivables, err := uc.payments.GetReceivablesByCustomer(ctx, companyID, s.CustomerID)
	if err != nil {
		return nil
	}

	now := time.Now()
	var outstanding float64
	for _, r := range receivables {
		outstanding += r.Balance
		if r.DueDate != nil && r.DueDate.Before(now) {
			return fmt.Errorf("%w: factura %s vencida desde %s (saldo %.2f)",
				domain.ErrOverdueBalance, r.SaleNumber, r.DueDate.Format("2006-01-02"), r.Balance)
		}
	}

	if customer.CreditLimit == nil {
		return nil
	}
	_, _, newTotal := s.GrandTotal()
	if outstanding+newTotal > *customer.CreditLimit {
		return fmt.Errorf("%w: cartera actual %.2f + esta venta %.2f superan el cupo de %.2f",
			domain.ErrCreditLimitExceeded, outstanding, newTotal, *customer.CreditLimit)
	}
	return nil
}
