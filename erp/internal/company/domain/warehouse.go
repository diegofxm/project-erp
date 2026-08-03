package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Warehouse representa una bodega o punto de almacenamiento de la empresa.
type Warehouse struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	// IsDefault — la bodega que usan las ventas/compras cuando el documento no elige una
	// explícitamente. Solo una por empresa (ver WarehouseRepository.SetDefault).
	IsDefault bool      `json:"is_default"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WarehouseRepository gestiona la persistencia de bodegas.
type WarehouseRepository interface {
	Save(ctx context.Context, w Warehouse) (*Warehouse, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Warehouse, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Warehouse, error)
	Update(ctx context.Context, w Warehouse) (*Warehouse, error)
	Deactivate(ctx context.Context, companyID, id uuid.UUID) error
	// SetDefault marca `id` como bodega por defecto y desmarca cualquier otra de la empresa.
	SetDefault(ctx context.Context, companyID, id uuid.UUID) error
	// GetOrCreateDefault devuelve la bodega por defecto de la empresa. Si ninguna está marcada
	// como tal, promueve la primera activa; si la empresa no tiene ninguna bodega todavía,
	// crea una "Principal" automáticamente — así sales/purchase siempre tienen dónde mover
	// stock sin exigirle al usuario configurar bodegas antes de poder vender o comprar.
	GetOrCreateDefault(ctx context.Context, companyID uuid.UUID) (*Warehouse, error)
}

var (
	ErrWarehouseNotFound  = errors.New("bodega no encontrada")
	ErrWarehouseCodeTaken = errors.New("ya existe una bodega con ese código")
)
