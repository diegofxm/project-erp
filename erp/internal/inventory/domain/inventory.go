package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MovementType clasifica cada movimiento de inventario.
type MovementType string

const (
	MovementEntry    MovementType = "entry"    // entrada: compra, devolución venta
	MovementExit     MovementType = "exit"     // salida: venta, pérdida
	MovementTransfer MovementType = "transfer" // traslado entre bodegas
	MovementAdjust   MovementType = "adjust"   // ajuste manual
)

// StockEntry representa el stock disponible de un producto en una bodega.
type StockEntry struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	ProductID uuid.UUID
	Warehouse string  // código o nombre de la bodega
	Quantity  float64 // puede ser fraccionario (kg, litros, etc.)
	UpdatedAt time.Time
}

// Movement registra cada cambio de stock (entrada, salida, ajuste).
type Movement struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	ProductID   uuid.UUID
	Warehouse   string
	Type        MovementType
	Quantity    float64 // siempre positivo; el tipo define si suma o resta
	Reference   string  // número de factura, OC, etc.
	Description string
	CreatedAt   time.Time
}

var (
	ErrInsufficientStock = errors.New("stock insuficiente")
	ErrStockNotFound     = errors.New("stock no encontrado")
	ErrMovementNotFound  = errors.New("movimiento no encontrado")
)
