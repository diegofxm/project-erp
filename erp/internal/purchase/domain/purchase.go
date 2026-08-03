package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type PurchaseStatus string

const (
	StatusDraft     PurchaseStatus = "draft"
	StatusConfirmed PurchaseStatus = "confirmed"
	StatusReceived  PurchaseStatus = "received"
	StatusCancelled PurchaseStatus = "cancelled"
)

// PurchaseOrder es una orden de compra a un proveedor.
type PurchaseOrder struct {
	ID         uuid.UUID
	CompanyID  uuid.UUID
	SupplierID uuid.UUID
	Number     string
	Status     PurchaseStatus
	IssueDate  time.Time
	DueDate    *time.Time // fecha esperada de recepción
	Notes      string
	Lines      []PurchaseLine
	// SupportDocumentID — Documento Soporte ya generado desde esta orden, si alguno
	// (electronic.documents.id, sin FK — cada módulo es dueño de su schema). nil = todavía no
	// se ha generado. Evita generar dos veces desde la misma orden (ver electronic
	// CreateFromPurchaseUseCase).
	SupportDocumentID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type PurchaseLine struct {
	ID              uuid.UUID
	PurchaseOrderID uuid.UUID
	ProductID       uuid.UUID
	Description     string
	Quantity        float64
	UnitPrice       float64
	Discount        float64 // porcentaje 0-100, aplicado antes de impuestos
	TaxRate         float64
	Subtotal        float64 // Quantity * UnitPrice - descuento
	TaxAmount       float64 // Subtotal * TaxRate / 100
	Total           float64 // Subtotal + TaxAmount
}

func (o *PurchaseOrder) CalculateTotals() {
	for i := range o.Lines {
		l := &o.Lines[i]
		gross := l.Quantity * l.UnitPrice
		l.Subtotal = gross - gross*l.Discount/100
		l.TaxAmount = l.Subtotal * l.TaxRate / 100
		l.Total = l.Subtotal + l.TaxAmount
	}
}

func (o *PurchaseOrder) GrandTotal() float64 {
	var t float64
	for _, l := range o.Lines {
		t += l.Total
	}
	return t
}

// PurchaseReceived se publica cuando la OC pasa a estado "recibida".
// inventory/ y accounting/ lo consumen.
type PurchaseReceived struct {
	PurchaseID uuid.UUID
	CompanyID  uuid.UUID
	SupplierID uuid.UUID
	Total      float64
	TaxAmount  float64
	IssueDate  time.Time
	Lines      []PurchaseLine
}

func (PurchaseReceived) EventName() string { return "purchase.received" }

var (
	ErrPurchaseNotFound     = errors.New("orden de compra no encontrada")
	ErrPurchaseNotDraft     = errors.New("la orden debe estar en borrador para esta operación")
	ErrPurchaseNotConfirmed = errors.New("la orden debe estar confirmada para recibir mercancía")
)
