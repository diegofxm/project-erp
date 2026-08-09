package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	log        *slog.Logger
}

func NewOnSaleConfirmed(repo domain.Repository, products productdomain.Repository, warehouses companydomain.WarehouseRepository, log *slog.Logger) *OnSaleConfirmed {
	return &OnSaleConfirmed{repo: repo, products: products, warehouses: warehouses, log: log}
}

func (h *OnSaleConfirmed) Register(bus *events.Bus) {
	bus.Subscribe(salesdomain.SaleConfirmed{}.EventName(), func(ctx context.Context, evt events.Event) error {
		ev, ok := evt.(salesdomain.SaleConfirmed)
		if !ok {
			return nil
		}
		if err := h.handle(ctx, ev); err != nil {
			h.log.Error("descontar stock venta", "sale_id", ev.SaleID, "error", err)
			return fmt.Errorf("inventory: descontar stock venta %s: %w", ev.SaleID, err)
		}
		return nil
	})
}

func (h *OnSaleConfirmed) handle(ctx context.Context, ev salesdomain.SaleConfirmed) error {
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
			errs = append(errs, fmt.Errorf("producto %s: %w", line.ProductID, err))
		}
	}
	return errors.Join(errs...)
}
