package assets

import (
	"context"

	"github.com/google/uuid"
)

// Repository abstrae el acceso a datos del módulo de activos fijos.
type Repository interface {
	// Create registra un nuevo activo fijo.
	Create(ctx context.Context, asset FixedAsset) (*FixedAsset, error)

	// GetByID devuelve un activo por su UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*FixedAsset, error)

	// ListByCompany devuelve todos los activos de una empresa.
	// Si activeOnly = true, solo devuelve los que están en estado ACTIVE.
	ListByCompany(ctx context.Context, companyID uuid.UUID, activeOnly bool) ([]*FixedAsset, error)

	// UpdateStatus cambia el estado de un activo (ACTIVE → DISPOSED / FULLY_DEPRECIATED).
	UpdateStatus(ctx context.Context, id uuid.UUID, status AssetStatus) error

	// GetAccumulatedDepreciation devuelve la suma de las entradas de depreciación
	// completadas para un activo (en centavos).
	GetAccumulatedDepreciation(ctx context.Context, assetID uuid.UUID) (int64, error)

	// GetRunForPeriod devuelve la corrida completada para (companyID, periodID),
	// o ErrAlreadyDepreciated si existe.
	GetRunForPeriod(ctx context.Context, companyID, periodID uuid.UUID) (*DepreciationRun, error)

	// CreateRun persiste la corrida y sus entradas por activo en una transacción.
	CreateRun(ctx context.Context, run DepreciationRun, entries []DepreciationEntry) (*DepreciationRun, error)
}
