package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/inventory/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

// OnPurchaseReceived ingresa al inventario cada línea de la OC recibida. Los productos de
// servicio (IsService) no llevan inventario, se excluyen — mismo criterio que
// OnSaleConfirmed (antes generaban entradas de stock sin sentido para "productos" que en
// realidad son servicios).
type OnPurchaseReceived struct {
	repo       domain.Repository
	products   productdomain.Repository
	warehouses companydomain.WarehouseRepository
	log        *slog.Logger
}

func NewOnPurchaseReceived(repo domain.Repository, products productdomain.Repository, warehouses companydomain.WarehouseRepository, log *slog.Logger) *OnPurchaseReceived {
	return &OnPurchaseReceived{repo: repo, products: products, warehouses: warehouses, log: log}
}

func (h *OnPurchaseReceived) Register(bus *events.Bus) {
	bus.Subscribe(purchasedomain.PurchaseReceived{}.EventName(), func(ctx context.Context, evt events.Event) error {
		ev, ok := evt.(purchasedomain.PurchaseReceived)
		if !ok {
			return nil
		}
		if err := h.handle(ctx, ev); err != nil {
			h.log.Error("entrada stock compra", "purchase_id", ev.PurchaseID, "error", err)
			return fmt.Errorf("inventory: entrada stock compra %s: %w", ev.PurchaseID, err)
		}
		return nil
	})
}

func (h *OnPurchaseReceived) handle(ctx context.Context, ev purchasedomain.PurchaseReceived) error {
	warehouse, err := h.warehouses.GetOrCreateDefault(ctx, ev.CompanyID)
	if err != nil {
		return err
	}
	// Se acumulan los errores por línea en vez de tragarlos -- antes de este cambio un fallo acá
	// nunca llegaba al Register de arriba (handle siempre devolvía nil), así que aunque el punto
	// 10 ya propaga errores del bus, esta función en particular nunca tenía ninguno que propagar.
	var errs []error
	for _, line := range ev.Lines {
		if p, err := h.products.GetByID(ctx, ev.CompanyID, line.ProductID); err == nil && p.IsService {
			continue
		}
		m := domain.Movement{
			CompanyID:   ev.CompanyID,
			ProductID:   line.ProductID,
			WarehouseID: warehouse.ID,
			Type:        domain.MovementEntry,
			Quantity:    line.Quantity,
			Reference:   ev.Number,
			Description: "Entrada por compra",
		}
		if err := applyMovement(ctx, h.repo, m); err != nil {
			errs = append(errs, fmt.Errorf("producto %s: %w", line.ProductID, err))
		}
	}
	return errors.Join(errs...)
}
