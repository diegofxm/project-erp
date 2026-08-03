// Package stock implementa notification/domain.StockPort leyendo desde los repositorios de
// inventory, product y company (bodegas) — mismo patrón que infrastructure/numbering.
package stock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	inventorydomain "github.com/diegofxm/erp/internal/inventory/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	"github.com/diegofxm/erp/internal/shared/notification/domain"
)

// Adapter implementa domain.StockPort combinando stock actual + punto de reorden del producto.
type Adapter struct {
	inventory  inventorydomain.Repository
	products   productdomain.Repository
	warehouses companydomain.WarehouseRepository
}

// New crea el adaptador.
func New(inventory inventorydomain.Repository, products productdomain.Repository, warehouses companydomain.WarehouseRepository) *Adapter {
	return &Adapter{inventory: inventory, products: products, warehouses: warehouses}
}

var _ domain.StockPort = (*Adapter)(nil)

// ListLowStock recorre todo el stock de la empresa y filtra las filas por debajo del mínimo
// configurado en el producto (MinStock=0 significa "sin umbral", se ignora).
func (a *Adapter) ListLowStock(ctx context.Context, companyID uuid.UUID) ([]domain.StockInfo, error) {
	entries, err := a.inventory.ListStock(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("stock adapter: listar stock: %w", err)
	}

	var out []domain.StockInfo
	for _, e := range entries {
		p, err := a.products.GetByID(ctx, companyID, e.ProductID)
		if err != nil || p.MinStock <= 0 || e.Quantity >= p.MinStock {
			continue
		}
		warehouseName := ""
		if w, err := a.warehouses.GetByID(ctx, companyID, e.WarehouseID); err == nil {
			warehouseName = w.Name
		}
		out = append(out, domain.StockInfo{
			ProductID: e.ProductID, ProductName: p.Name,
			WarehouseID: e.WarehouseID, WarehouseName: warehouseName,
			Quantity: e.Quantity, MinStock: p.MinStock,
		})
	}
	return out, nil
}
