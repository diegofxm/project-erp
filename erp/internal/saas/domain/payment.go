package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PaymentType string

const (
	PaymentPlan        PaymentType = "plan"        // cobro del plan (afiliación o renovación)
	PaymentCertificate PaymentType = "certificate" // certificado DIAN vendido por nosotros
	PaymentOverage     PaymentType = "overage"     // documentos por encima del cupo incluido
)

// Payment es un registro manual de cobro — no hay pasarela de pago integrada todavía, lo anota el
// superadmin (mismo criterio que el sistema legado, ver _legacy/apidian/internal/subscriptions).
type Payment struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	SubscriptionID *uuid.UUID
	Type           PaymentType
	AmountCents    int64
	Note           string
	PaidAt         time.Time
	CreatedAt      time.Time
}

type PaymentRepository interface {
	Create(ctx context.Context, p Payment) (*Payment, error)
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]Payment, error)
}
