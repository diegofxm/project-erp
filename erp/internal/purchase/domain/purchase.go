package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	cofdom "github.com/diegofxm/cofacture/domain"
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
	// PaymentMeans -- mismo tipo/catálogos DIAN que Sale.PaymentMeans y que electronic; se
	// hereda al Documento Soporte generado desde esta orden (ver electronic/application/
	// from_purchase.go, que antes forzaba "Contado/Efectivo" sin importar lo pactado).
	PaymentMeans []cofdom.PaymentMean
	// SupportDocumentID — Documento Soporte ya generado desde esta orden, si alguno
	// (electronic.documents.id, sin FK — cada módulo es dueño de su schema). nil = todavía no
	// se ha generado. Evita generar dos veces desde la misma orden (ver electronic
	// CreateFromPurchaseUseCase).
	SupportDocumentID *uuid.UUID
	Withholdings      []PurchaseWithholding
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// PurchaseWithholding es una retención (ReteFuente/ReteIVA/ReteICA) aplicada a la orden antes
// de recibirla — reduce lo que se le paga al proveedor y genera un pasivo por retención a
// favor de la DIAN. ConceptName, RateBP y AccountPayable son una foto del concepto (accounting.
// withholding_concepts) al momento de aplicarla — si el concepto cambia después, esta orden no
// se ve afectada retroactivamente.
type PurchaseWithholding struct {
	ID              uuid.UUID
	PurchaseOrderID uuid.UUID
	ConceptCode     string
	ConceptName     string
	Base            float64
	RateBP          int
	Amount          float64
	AccountPayable  string
	CreatedAt       time.Time
}

// TotalWithholding suma las retenciones aplicadas a la orden.
func (o *PurchaseOrder) TotalWithholding() float64 {
	var t float64
	for _, w := range o.Withholdings {
		t += w.Amount
	}
	return t
}

// NetPayable es lo que realmente se le paga al proveedor: total de la orden menos retenciones.
func (o *PurchaseOrder) NetPayable() float64 {
	return o.GrandTotal() - o.TotalWithholding()
}

type PurchaseLine struct {
	ID              uuid.UUID
	PurchaseOrderID uuid.UUID
	ProductID       uuid.UUID
	Description     string
	// UnitCode -- ver el mismo campo en sales.domain.SaleLine.
	UnitCode  string
	Quantity  float64
	UnitPrice float64
	Discount  float64 // porcentaje 0-100, aplicado antes de impuestos
	TaxRate   float64
	Subtotal  float64 // Quantity * UnitPrice - descuento
	TaxAmount float64 // Subtotal * TaxRate / 100
	Total     float64 // Subtotal + TaxAmount
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
	PurchaseID   uuid.UUID
	CompanyID    uuid.UUID
	SupplierID   uuid.UUID
	Number       string
	Total        float64
	TaxAmount    float64
	IssueDate    time.Time
	Lines        []PurchaseLine
	Withholdings []PurchaseWithholding
}

func (PurchaseReceived) EventName() string { return "purchase.received" }

var (
	ErrPurchaseNotFound     = errors.New("orden de compra no encontrada")
	ErrPurchaseNotDraft     = errors.New("la orden debe estar en borrador para esta operación")
	ErrPurchaseNotConfirmed = errors.New("la orden debe estar confirmada para recibir mercancía")
	// ErrNumberCounterInvalid / ErrNumberCounterBackwards — al fijar manualmente el consecutivo de
	// orden de compra (ver application/number_counter.go).
	ErrNumberCounterInvalid   = errors.New("el próximo número debe ser mayor o igual a 1")
	ErrNumberCounterBackwards = errors.New("el consecutivo indicado ya fue superado — no se puede retroceder")
)
