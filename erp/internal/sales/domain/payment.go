package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PaymentMethod identifica el medio de pago.
type PaymentMethod string

const (
	PaymentCash     PaymentMethod = "cash"
	PaymentTransfer PaymentMethod = "transfer"
	PaymentCheck    PaymentMethod = "check"
	PaymentCard     PaymentMethod = "card"
	PaymentOther    PaymentMethod = "other"
)

// SalePayment registra un pago recibido contra una venta confirmada.
type SalePayment struct {
	ID            uuid.UUID
	CompanyID     uuid.UUID
	SaleID        uuid.UUID
	PaymentDate   time.Time
	Amount        float64
	PaymentMethod PaymentMethod
	Reference     string
	Notes         string
	CreatedAt     time.Time
}

// ReceivableBalance agrupa el saldo pendiente de una venta.
type ReceivableBalance struct {
	SaleID     uuid.UUID
	SaleNumber string
	CustomerID uuid.UUID
	IssueDate  time.Time
	DueDate    *time.Time
	Total      float64
	Paid       float64
	Balance    float64
}

// PaymentRepository gestiona pagos y saldos de cartera.
type PaymentRepository interface {
	Save(ctx context.Context, p SalePayment) (*SalePayment, error)
	ListBySale(ctx context.Context, companyID, saleID uuid.UUID) ([]SalePayment, error)
	GetReceivables(ctx context.Context, companyID uuid.UUID) ([]ReceivableBalance, error)
}

var ErrPaymentNotFound = errors.New("pago no encontrado")
