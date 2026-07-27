package events

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceConfirmed se emite cuando Sales confirma una factura electrónica.
// Suscriptores: Accounting (asiento), Inventory (descuento de stock), Electronic (envío DIAN).
type InvoiceConfirmed struct {
	InvoiceID        uuid.UUID
	CompanyID        uuid.UUID
	NumberingRangeID uuid.UUID
	ConfirmedAt      time.Time
}

func (InvoiceConfirmed) EventName() string { return "invoice.confirmed" }

// InvoiceCancelled se emite cuando Sales anula una factura.
// Suscriptores: Accounting (reversa del asiento), Inventory (reingreso de stock).
type InvoiceCancelled struct {
	InvoiceID   uuid.UUID
	CompanyID   uuid.UUID
	CancelledAt time.Time
}

func (InvoiceCancelled) EventName() string { return "invoice.cancelled" }

// PaymentReceived se emite cuando Sales registra un pago de cliente.
// Suscriptores: Accounting (abono a cartera).
type PaymentReceived struct {
	PaymentID   uuid.UUID
	CompanyID   uuid.UUID
	CustomerID  uuid.UUID
	AmountCents int64
	ReceivedAt  time.Time
}

func (PaymentReceived) EventName() string { return "payment.received" }

// PurchaseReceived se emite cuando Purchase registra la recepción de mercancía.
// Suscriptores: Inventory (ingreso de stock), Accounting (asiento de compra).
type PurchaseReceived struct {
	PurchaseID  uuid.UUID
	CompanyID   uuid.UUID
	SupplierID  uuid.UUID
	ReceivedAt  time.Time
}

func (PurchaseReceived) EventName() string { return "purchase.received" }

// StockMoved se emite cuando Inventory registra un movimiento de mercancía.
// Suscriptores: Accounting (asiento de costo de ventas / traslado).
type StockMoved struct {
	MovementID uuid.UUID
	CompanyID  uuid.UUID
	ProductID  uuid.UUID
	Quantity   int64
	MovedAt    time.Time
}

func (StockMoved) EventName() string { return "stock.moved" }

// PayrollGenerated se emite cuando Payroll cierra un período de nómina.
// Suscriptores: Accounting (asiento de nómina), Electronic (nómina electrónica DIAN).
type PayrollGenerated struct {
	PayrollID   uuid.UUID
	CompanyID   uuid.UUID
	Period      string // "2026-01"
	GeneratedAt time.Time
}

func (PayrollGenerated) EventName() string { return "payroll.generated" }

// OrderConfirmed se emite cuando Sales confirma un pedido de cliente.
// Suscriptores: Inventory (reserva de stock).
type OrderConfirmed struct {
	OrderID     uuid.UUID
	CompanyID   uuid.UUID
	ConfirmedAt time.Time
}

func (OrderConfirmed) EventName() string { return "order.confirmed" }
