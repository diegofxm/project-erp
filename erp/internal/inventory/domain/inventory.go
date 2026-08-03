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
// WarehouseID referencia company.warehouses.id — sin FK a nivel de base de datos porque cada
// módulo es dueño de su propio schema (mismo criterio que sales.sales.invoice_document_id),
// la integridad se garantiza en la aplicación.
type StockEntry struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	ProductID   uuid.UUID
	WarehouseID uuid.UUID
	Quantity    float64 // puede ser fraccionario (kg, litros, etc.)
	UpdatedAt   time.Time
}

// Movement registra cada cambio de stock (entrada, salida, ajuste, traslado).
// Una transferencia entre bodegas genera DOS movimientos (salida en origen + entrada en
// destino) que comparten el mismo TransferGroupID — ver Repository.Transfer.
type Movement struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	ProductID       uuid.UUID
	WarehouseID     uuid.UUID
	Type            MovementType
	Quantity        float64 // siempre positivo; el tipo define si suma o resta
	Reference       string  // número de factura, OC, etc.
	Description     string
	TransferGroupID *uuid.UUID // solo en movimientos tipo transfer — enlaza el par salida/entrada
	CreatedAt       time.Time
}

var (
	ErrInsufficientStock = errors.New("stock insuficiente")
	ErrStockNotFound     = errors.New("stock no encontrado")
	ErrMovementNotFound  = errors.New("movimiento no encontrado")
)
