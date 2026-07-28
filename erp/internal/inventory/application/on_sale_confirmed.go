package application

import (
	"context"
	"log"

	"github.com/diegofxm/erp/internal/inventory/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnSaleConfirmed descuenta del inventario cada línea de la venta confirmada.
type OnSaleConfirmed struct {
	repo domain.Repository
}

func NewOnSaleConfirmed(repo domain.Repository) *OnSaleConfirmed {
	return &OnSaleConfirmed{repo: repo}
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
	for _, line := range ev.Lines {
		m := domain.Movement{
			CompanyID:   ev.CompanyID,
			ProductID:   line.ProductID,
			Warehouse:   "principal",
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
