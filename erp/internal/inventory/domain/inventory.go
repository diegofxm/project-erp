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

// NumberPrefix devuelve el prefijo de folio para el tipo de movimiento (ver
// inventory.number_counters, un contador por company_id + doc_type + año).
func (t MovementType) NumberPrefix() string {
	switch t {
	case MovementEntry:
		return "ENT"
	case MovementExit:
		return "SAL"
	case MovementTransfer:
		return "TRA"
	case MovementAdjust:
		return "AJU"
	default:
		return "MOV"
	}
}

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
	Number          string // folio interno, ej. "ENT-2026-00001" — asignado por el repositorio al guardar
	ProductID       uuid.UUID
	WarehouseID     uuid.UUID
	Type            MovementType
	Quantity        float64 // siempre positivo; el tipo define si suma o resta
	Reference       string  // número de factura, OC, etc.
	Description     string
	TransferGroupID *uuid.UUID // solo en movimientos tipo transfer — enlaza el par salida/entrada
	// IsAddition indica si este movimiento sumó (true) o restó (false) al stock -- necesario para
	// poder revertir el efecto al eliminar (ver DeleteMovement), ya que Type solo no alcanza para
	// distinguir el lado origen/destino de un transfer.
	IsAddition bool
	CreatedAt  time.Time
}

// StockDelta es el efecto que este movimiento tuvo sobre el stock, con signo.
func (m Movement) StockDelta() float64 {
	if m.IsAddition {
		return m.Quantity
	}
	return -m.Quantity
}

var (
	ErrInsufficientStock = errors.New("stock insuficiente")
	ErrStockNotFound     = errors.New("stock no encontrado")
	ErrMovementNotFound  = errors.New("movimiento no encontrado")
	// ErrDeleteWouldMakeStockNegative: revertir este movimiento dejaría el stock en negativo --
	// típicamente porque ya se registraron salidas posteriores que dependían de esta entrada.
	// Hay que resolver/eliminar esos movimientos posteriores primero.
	ErrDeleteWouldMakeStockNegative = errors.New("eliminar este movimiento dejaría el stock en negativo -- revisa los movimientos posteriores del mismo producto/bodega")
)
