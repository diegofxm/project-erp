package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type SaleStatus string

const (
	StatusDraft     SaleStatus = "draft"
	StatusConfirmed SaleStatus = "confirmed"
	StatusCancelled SaleStatus = "cancelled"
)

// Sale es una factura de venta (borrador → confirmada → cancelada).
// Al confirmar se publica SaleConfirmed para que electronic/ genere el documento DIAN
// y accounting/ registre el asiento.
type Sale struct {
	ID         uuid.UUID
	CompanyID  uuid.UUID
	CustomerID uuid.UUID
	Number     string     // número interno (consecutivo)
	Status     SaleStatus
	IssueDate  time.Time
	DueDate    *time.Time // fecha vencimiento cartera
	Notes      string
	Lines      []SaleLine
	// InvoiceDocumentID — factura electrónica ya generada desde esta venta, si alguna
	// (electronic.documents.id, sin FK — cada módulo es dueño de su schema). nil = todavía no
	// se ha facturado.
	InvoiceDocumentID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SaleLine struct {
	ID          uuid.UUID
	SaleID      uuid.UUID
	ProductID   uuid.UUID
	Description string
	Quantity    float64
	UnitPrice   float64
	Discount    float64 // porcentaje 0-100, aplicado antes de impuestos
	TaxRate     float64
	Subtotal    float64
	TaxAmount   float64
	Total       float64
}

// SaleConfirmed es el evento publicado al confirmar una venta.
// accounting/ e inventory/ lo consumen.
type SaleConfirmed struct {
	SaleID     uuid.UUID
	CompanyID  uuid.UUID
	CustomerID uuid.UUID
	Number     string
	Total      float64
	TaxAmount  float64
	IssueDate  time.Time
	Lines      []SaleLine
}

func (SaleConfirmed) EventName() string { return "sale.confirmed" }

func (s *Sale) CalculateTotals() {
	for i := range s.Lines {
		l := &s.Lines[i]
		gross := l.Quantity * l.UnitPrice
		l.Subtotal = gross - gross*l.Discount/100
		l.TaxAmount = l.Subtotal * l.TaxRate / 100
		l.Total = l.Subtotal + l.TaxAmount
	}
}

func (s *Sale) GrandTotal() (subtotal, tax, total float64) {
	for _, l := range s.Lines {
		subtotal += l.Subtotal
		tax += l.TaxAmount
		total += l.Total
	}
	return
}

var (
	ErrSaleNotFound     = errors.New("venta no encontrada")
	ErrSaleNotDraft     = errors.New("la venta debe estar en borrador para esta operación")
	ErrSaleNotConfirmed = errors.New("la venta ya está confirmada o cancelada")
	// ErrInsufficientStock — el mensaje concreto (qué producto, cuánto falta) va en el wrap del
	// caller (ver application/confirm.go), esto solo sirve para que el handler HTTP lo detecte
	// con errors.Is y devuelva 422 en vez de 500.
	ErrInsufficientStock = errors.New("stock insuficiente para confirmar la venta")
	// ErrCreditLimitExceeded / ErrOverdueBalance — control de cartera al confirmar una venta a
	// crédito (ver application/confirm.go). Mismo criterio: mensaje detallado en el wrap.
	ErrCreditLimitExceeded = errors.New("la venta supera el cupo de crédito disponible del cliente")
	ErrOverdueBalance      = errors.New("el cliente tiene cartera vencida — no se puede confirmar una venta nueva")
	// ErrNumberCounterInvalid / ErrNumberCounterBackwards — al fijar manualmente el consecutivo de
	// venta o cotización (ver application/number_counter.go, para migrar empresas que ya traían
	// numeración propia de otro sistema).
	ErrNumberCounterInvalid   = errors.New("el próximo número debe ser mayor o igual a 1")
	ErrNumberCounterBackwards = errors.New("el consecutivo indicado ya fue superado — no se puede retroceder")
)
