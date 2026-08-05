package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BillingCycle string

const (
	BillingMonthly BillingCycle = "monthly"
	BillingAnnual  BillingCycle = "annual"
	BillingNone    BillingCycle = "none" // plan gratis, sin ciclo de cobro
)

// Plan es un paquete vendible: precio recurrente + cupo de documentos + variante de certificado +
// módulos que desbloquea. El certificado DIAN se factura siempre en ciclo anual propio,
// independiente del BillingCycle del plan (ver Subscription.CertExpiresAt) — así funciona en la
// realidad: el certificado se compra/renueva una vez al año sin importar si el plan de software
// es mensual o anual.
type Plan struct {
	ID          uuid.UUID
	Code        string
	Name        string
	Description string

	BillingCycle BillingCycle
	PriceCents   int64

	// IncludedDocuments nil = ilimitado. Si no es nil, el excedente se cobra (no bloquea) a
	// PricePerExtraDocumentCents por unidad.
	IncludedDocuments          *int
	PricePerExtraDocumentCents int64

	RequiresCertificate   bool
	CertificatePriceCents int64 // solo aplica si RequiresCertificate=true y la empresa no trae el propio

	// AnnualIncrementPct — porcentaje aplicado al precio contratado en la próxima renovación (no
	// retroactivo a suscripciones ya vigentes), ej. 5.5 = 5.5%. Se dispara manualmente desde el
	// panel admin (ver PlanUseCase.ApplyIncrement), igual que en el sistema legado.
	AnnualIncrementPct float64

	// IsInternal marca el plan usado por la empresa operadora de la plataforma (Cofacture) —
	// ilimitado, $0, excluido del catálogo público que ve un cliente nuevo.
	IsInternal bool
	IsActive   bool

	// ModuleCodes — módulos que este plan desbloquea (join saas.plan_modules), cargado junto al
	// plan por el repositorio.
	ModuleCodes []string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PlanRepository interface {
	Create(ctx context.Context, p Plan) (*Plan, error)
	Update(ctx context.Context, p Plan) (*Plan, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Plan, error)
	// List incluye planes internos — el filtrado para catálogo público lo hace el caller.
	List(ctx context.Context) ([]Plan, error)
}
