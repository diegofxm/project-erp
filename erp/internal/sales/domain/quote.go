package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	cofdom "github.com/diegofxm/cofacture/domain"
)

type QuoteStatus string

const (
	QuoteStatusDraft    QuoteStatus = "draft"
	QuoteStatusSent     QuoteStatus = "sent"
	QuoteStatusAccepted QuoteStatus = "accepted"
	QuoteStatusRejected QuoteStatus = "rejected"
	QuoteStatusExpired  QuoteStatus = "expired"
)

// Quote es una cotización que puede convertirse en venta al ser aceptada.
type Quote struct {
	ID         uuid.UUID
	CompanyID  uuid.UUID
	CustomerID uuid.UUID
	Number     string
	Status     QuoteStatus
	IssueDate  time.Time
	ValidUntil *time.Time
	Notes      string
	Lines      []QuoteLine
	// PaymentMeans -- mismo tipo/catálogos que Sale.PaymentMeans y que electronic; se hereda a
	// la venta al convertir (ver QuoteUseCase.ConvertToSale) y de ahí a la factura electrónica.
	PaymentMeans []cofdom.PaymentMean
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type QuoteLine struct {
	ID          uuid.UUID
	QuoteID     uuid.UUID
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

func (q *Quote) CalculateTotals() {
	for i := range q.Lines {
		l := &q.Lines[i]
		gross := l.Quantity * l.UnitPrice
		l.Subtotal = gross - gross*l.Discount/100
		l.TaxAmount = l.Subtotal * l.TaxRate / 100
		l.Total = l.Subtotal + l.TaxAmount
	}
}

func (q *Quote) GrandTotal() (subtotal, tax, total float64) {
	for _, l := range q.Lines {
		subtotal += l.Subtotal
		tax += l.TaxAmount
		total += l.Total
	}
	return
}

// QuoteRepository gestiona la persistencia de cotizaciones.
type QuoteRepository interface {
	Save(ctx context.Context, q Quote) (*Quote, error)
	// Update reemplaza cliente/fecha de validez/notas/líneas de una cotización EN BORRADOR --
	// devuelve ErrQuoteNotDraft si ya se envió (a partir de "sent" se corrige rechazando y
	// creando una nueva, no editando la ya enviada al cliente).
	Update(ctx context.Context, companyID, id uuid.UUID, q Quote) (*Quote, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Quote, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Quote, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status QuoteStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	// NextQuoteNumber devuelve el siguiente consecutivo de cotización para la empresa y el año
	// dados (arranca en 1 cada año) — comparte la tabla de contadores con NextSaleNumber
	// (sales.number_counters), distinguidas por doc_type.
	NextQuoteNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error)
	// SetQuoteNumberCounter — ver Repository.SetSaleNumberCounter, mismo comportamiento para
	// cotizaciones.
	SetQuoteNumberCounter(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error)
}

var (
	ErrQuoteNotFound    = errors.New("cotización no encontrada")
	ErrQuoteNotDraft    = errors.New("la cotización debe estar en borrador para esta operación")
	ErrQuoteNotSent     = errors.New("la cotización debe estar enviada para aceptar o rechazar")
	ErrQuoteNotAccepted = errors.New("solo las cotizaciones aceptadas pueden convertirse en venta")
)
