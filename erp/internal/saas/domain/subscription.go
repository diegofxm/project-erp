package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string

const (
	SubscriptionActive    SubscriptionStatus = "active"
	SubscriptionCancelled SubscriptionStatus = "cancelled"
	SubscriptionSuspended SubscriptionStatus = "suspended"
)

// Subscription vincula una empresa a un Plan. ContractedPriceCents es una foto del precio al
// contratar/renovar — si el plan sube de precio después (Plan.AnnualIncrementPct aplicado), las
// suscripciones ya vigentes no cambian de precio hasta su propia renovación.
type Subscription struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	PlanID    uuid.UUID

	// HasOwnCertificate — si la empresa trae su propio certificado DIAN (no se le cobra
	// Plan.CertificatePriceCents) o si se lo vendemos nosotros. Solo relevante cuando
	// Plan.RequiresCertificate=true.
	HasOwnCertificate bool

	Status               SubscriptionStatus
	ContractedPriceCents int64

	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time

	// CertExpiresAt — vencimiento del certificado DIAN, ciclo anual propio e independiente del
	// billing_cycle del plan. Nil si HasOwnCertificate=true (no lo gestionamos nosotros) o si el
	// plan no requiere certificado.
	CertExpiresAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s Subscription) (*Subscription, error)
	// GetActive devuelve la suscripción activa de la empresa, si tiene una — ErrSubscriptionNotFound
	// si no tiene ninguna (empresa sin plan asignado todavía).
	GetActive(ctx context.Context, companyID uuid.UUID) (*Subscription, error)
	// Cancel marca como cancelada la suscripción activa de la empresa (si tiene una), para poder
	// crear una nueva al asignar un plan distinto — solo una activa por empresa a la vez.
	Cancel(ctx context.Context, companyID uuid.UUID) error
	Renew(ctx context.Context, id uuid.UUID, newPeriodStart, newPeriodEnd time.Time, newContractedPriceCents int64) (*Subscription, error)
	// ListUpcomingRenewals devuelve las suscripciones activas cuyo CurrentPeriodEnd cae dentro de
	// los próximos withinDays días (incluye ya vencidas) — para el panel de renovaciones.
	ListUpcomingRenewals(ctx context.Context, withinDays int) ([]Subscription, error)
	// ListAllActive — para el resumen de facturación (todas las empresas con suscripción activa).
	ListAllActive(ctx context.Context) ([]Subscription, error)
}

var (
	ErrSubscriptionNotFound = errors.New("la empresa no tiene una suscripción activa")
	ErrPlanNotFound         = errors.New("plan no encontrado")
	ErrPlanInactive         = errors.New("el plan está desactivado")
)
