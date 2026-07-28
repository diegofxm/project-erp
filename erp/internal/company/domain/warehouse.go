package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Warehouse representa una bodega o punto de almacenamiento de la empresa.
type Warehouse struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Code      string
	Name      string
	Address   string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WarehouseRepository gestiona la persistencia de bodegas.
type WarehouseRepository interface {
	Save(ctx context.Context, w Warehouse) (*Warehouse, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Warehouse, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Warehouse, error)
	Update(ctx context.Context, w Warehouse) (*Warehouse, error)
	Deactivate(ctx context.Context, companyID, id uuid.UUID) error
}

var (
	ErrWarehouseNotFound  = errors.New("bodega no encontrada")
	ErrWarehouseCodeTaken = errors.New("ya existe una bodega con ese código")
)
