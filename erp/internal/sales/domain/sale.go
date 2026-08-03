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
	Total      float64
	TaxAmount  float64
	IssueDate  time.Time
	Lines      []SaleLine
}

func (SaleConfirmed) EventName() string { return "sale.confirmed" }

func (s *Sale) CalculateTotals() {
	for i := range s.Lines {
		l := &s.Lines[i]
		l.Subtotal = l.Quantity * l.UnitPrice
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
	ErrSaleNotFound    = errors.New("venta no encontrada")
	ErrSaleNotDraft    = errors.New("la venta debe estar en borrador para esta operación")
	ErrSaleNotConfirmed = errors.New("la venta ya está confirmada o cancelada")
)
