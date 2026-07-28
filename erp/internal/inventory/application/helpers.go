package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/inventory/domain"
)

// applyMovement graba el movimiento y actualiza el stock en una sola operación.
// Para salidas no verifica stock suficiente — el caller decide si es bloqueante.
func applyMovement(ctx context.Context, repo domain.Repository, m domain.Movement) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if _, err := repo.SaveMovement(ctx, m); err != nil {
		return err
	}

	current, err := repo.GetStock(ctx, m.CompanyID, m.ProductID, m.Warehouse)
	var currentQty float64
	if err == nil {
		currentQty = current.Quantity
	}

	var newQty float64
	switch m.Type {
	case domain.MovementEntry, domain.MovementAdjust:
		newQty = currentQty + m.Quantity
	default:
		newQty = currentQty - m.Quantity
	}

	entry := domain.StockEntry{
		ID:        uuid.New(),
		CompanyID: m.CompanyID,
		ProductID: m.ProductID,
		Warehouse: m.Warehouse,
		Quantity:  newQty,
	}
	if current != nil {
		entry.ID = current.ID
	}
	return repo.UpsertStock(ctx, entry)
}
