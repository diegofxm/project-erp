package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	// Stock
	GetStock(ctx context.Context, companyID, productID uuid.UUID, warehouse string) (*StockEntry, error)
	ListStock(ctx context.Context, companyID uuid.UUID) ([]StockEntry, error)
	UpsertStock(ctx context.Context, e StockEntry) error

	// Movimientos
	SaveMovement(ctx context.Context, m Movement) (*Movement, error)
	ListMovements(ctx context.Context, companyID uuid.UUID, productID *uuid.UUID) ([]Movement, error)
}
