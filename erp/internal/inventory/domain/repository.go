package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// Stock
	GetStock(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*StockEntry, error)
	ListStock(ctx context.Context, companyID uuid.UUID) ([]StockEntry, error)
	UpsertStock(ctx context.Context, e StockEntry) error

	// Movimientos
	SaveMovement(ctx context.Context, m Movement) (*Movement, error)
	ListMovements(ctx context.Context, companyID uuid.UUID, productID *uuid.UUID) ([]Movement, error)
	// DeleteMovement elimina un movimiento (o el par completo, si es un transfer) y revierte su
	// efecto sobre el stock. Devuelve ErrMovementNotFound si no existe o no pertenece a la
	// empresa, ErrDeleteWouldMakeStockNegative si revertirlo dejaría el stock en negativo.
	DeleteMovement(ctx context.Context, companyID, id uuid.UUID) error

	// Transfer traslada `quantity` de `fromWarehouseID` a `toWarehouseID` de forma atómica:
	// valida stock suficiente en origen, genera los dos movimientos (exit + entry) enlazados
	// por un TransferGroupID compartido, y actualiza ambos saldos — todo en una transacción.
	Transfer(ctx context.Context, companyID, productID, fromWarehouseID, toWarehouseID uuid.UUID, quantity float64, reference, description string) (fromMovement, toMovement *Movement, err error)
}
