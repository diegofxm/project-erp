package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
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
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type QuoteLine struct {
	ID          uuid.UUID
	QuoteID     uuid.UUID
	ProductID   uuid.UUID
	Description string
	Quantity    float64
	UnitPrice   float64
	TaxRate     float64
	Subtotal    float64
	TaxAmount   float64
	Total       float64
}

func (q *Quote) CalculateTotals() {
	for i := range q.Lines {
		l := &q.Lines[i]
		l.Subtotal = l.Quantity * l.UnitPrice
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
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Quote, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Quote, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status QuoteStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
}

var (
	ErrQuoteNotFound    = errors.New("cotización no encontrada")
	ErrQuoteNotDraft    = errors.New("la cotización debe estar en borrador para esta operación")
	ErrQuoteNotSent     = errors.New("la cotización debe estar enviada para aceptar o rechazar")
	ErrQuoteNotAccepted = errors.New("solo las cotizaciones aceptadas pueden convertirse en venta")
)
