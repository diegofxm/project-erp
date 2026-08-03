package application

import (
	"context"
	"log"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/inventory/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnSaleConfirmed descuenta del inventario cada línea de la venta confirmada. La disponibilidad
// ya se validó de forma síncrona y bloqueante en sales/application/confirm.go antes de llegar
// acá — este handler asíncrono solo aplica el movimiento, no vuelve a decidir si hay stock.
type OnSaleConfirmed struct {
	repo       domain.Repository
	products   productdomain.Repository
	warehouses companydomain.WarehouseRepository
}

func NewOnSaleConfirmed(repo domain.Repository, products productdomain.Repository, warehouses companydomain.WarehouseRepository) *OnSaleConfirmed {
	return &OnSaleConfirmed{repo: repo, products: products, warehouses: warehouses}
}

func (h *OnSaleConfirmed) Register(bus *events.Bus) {
	bus.Subscribe(salesdomain.SaleConfirmed{}.EventName(), func(evt events.Event) {
		ev, ok := evt.(salesdomain.SaleConfirmed)
		if !ok {
			return
		}
		ctx := context.Background()
		if err := h.handle(ctx, ev); err != nil {
			log.Printf("inventory: descontar stock venta %s: %v", ev.SaleID, err)
		}
	})
}

func (h *OnSaleConfirmed) handle(ctx context.Context, ev salesdomain.SaleConfirmed) error {
	warehouse, err := h.warehouses.GetOrCreateDefault(ctx, ev.CompanyID)
	if err != nil {
		return err
	}
	for _, line := range ev.Lines {
		if p, err := h.products.GetByID(ctx, ev.CompanyID, line.ProductID); err == nil && p.IsService {
			continue // los servicios no llevan inventario — nunca deben generar un movimiento
		}
		m := domain.Movement{
			CompanyID:   ev.CompanyID,
			ProductID:   line.ProductID,
			WarehouseID: warehouse.ID,
			Type:        domain.MovementExit,
			Quantity:    line.Quantity,
			Reference:   ev.SaleID.String(),
			Description: "Salida por venta",
		}
		if err := applyMovement(ctx, h.repo, m); err != nil {
			log.Printf("inventory: producto %s venta %s: %v", line.ProductID, ev.SaleID, err)
		}
	}
	return nil
}
