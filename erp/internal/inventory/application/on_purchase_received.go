package application

import (
	"context"
	"log"

	"github.com/diegofxm/erp/internal/inventory/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPurchaseReceived ingresa al inventario cada línea de la OC recibida.
type OnPurchaseReceived struct {
	repo domain.Repository
}

func NewOnPurchaseReceived(repo domain.Repository) *OnPurchaseReceived {
	return &OnPurchaseReceived{repo: repo}
}

func (h *OnPurchaseReceived) Register(bus *events.Bus) {
	bus.Subscribe(purchasedomain.PurchaseReceived{}.EventName(), func(evt events.Event) {
		ev, ok := evt.(purchasedomain.PurchaseReceived)
		if !ok {
			return
		}
		ctx := context.Background()
		if err := h.handle(ctx, ev); err != nil {
			log.Printf("inventory: entrada stock compra %s: %v", ev.PurchaseID, err)
		}
	})
}

func (h *OnPurchaseReceived) handle(ctx context.Context, ev purchasedomain.PurchaseReceived) error {
	for _, line := range ev.Lines {
		m := domain.Movement{
			CompanyID:   ev.CompanyID,
			ProductID:   line.ProductID,
			Warehouse:   "principal",
			Type:        domain.MovementEntry,
			Quantity:    line.Quantity,
			Reference:   ev.PurchaseID.String(),
			Description: "Entrada por compra",
		}
		if err := applyMovement(ctx, h.repo, m); err != nil {
			log.Printf("inventory: producto %s compra %s: %v", line.ProductID, ev.PurchaseID, err)
		}
	}
	return nil
}
