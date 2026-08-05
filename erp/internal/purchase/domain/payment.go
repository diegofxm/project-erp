package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PaymentMethod identifica el medio de pago — mismos valores que sales.PaymentMethod.
type PaymentMethod string

const (
	PaymentCash     PaymentMethod = "cash"
	PaymentTransfer PaymentMethod = "transfer"
	PaymentCheck    PaymentMethod = "check"
	PaymentCard     PaymentMethod = "card"
	PaymentOther    PaymentMethod = "other"
)

// PurchasePayment registra un pago realizado contra una orden de compra recibida.
type PurchasePayment struct {
	ID            uuid.UUID
	CompanyID     uuid.UUID
	PurchaseID    uuid.UUID
	PaymentDate   time.Time
	Amount        float64
	PaymentMethod PaymentMethod
	Reference     string
	Notes         string
	CreatedAt     time.Time
}

// PurchasePaymentRecorded se publica al registrar un pago sobre una orden recibida.
// accounting/ lo consume para reducir cuentas por pagar y disminuir caja/bancos.
type PurchasePaymentRecorded struct {
	PaymentID      uuid.UUID
	CompanyID      uuid.UUID
	PurchaseID     uuid.UUID
	PurchaseNumber string
	Amount         float64
	PaymentMethod  PaymentMethod
	PaymentDate    time.Time
}

func (PurchasePaymentRecorded) EventName() string { return "purchase.payment_recorded" }

// PayableBalance agrupa el saldo pendiente de una orden de compra recibida — espejo de
// sales.ReceivableBalance pero del lado de lo que nosotros debemos al proveedor.
type PayableBalance struct {
	PurchaseID     uuid.UUID
	PurchaseNumber string
	SupplierID     uuid.UUID
	IssueDate      time.Time
	DueDate        *time.Time
	Total          float64
	Paid           float64
	Balance        float64
}

// PaymentRepository gestiona pagos y saldos de cuentas por pagar.
type PaymentRepository interface {
	Save(ctx context.Context, p PurchasePayment) (*PurchasePayment, error)
	ListByPurchase(ctx context.Context, companyID, purchaseID uuid.UUID) ([]PurchasePayment, error)
	GetPayables(ctx context.Context, companyID uuid.UUID) ([]PayableBalance, error)
}

var ErrPaymentNotFound = errors.New("pago no encontrado")
