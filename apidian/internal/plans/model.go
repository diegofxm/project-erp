package plans

import (
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID                   uuid.UUID
	Name                 string
	Description          string
	MaxDocumentsPerMonth *int    // nil = ilimitado
	MaxIssuers           int
	PriceCOP             int     // tarifa mensual de suscripción (si aplica)
	AffiliationFeeCOP    int     // tarifa única de afiliación inicial
	RenewalFeeCOP        int     // tarifa de renovación anual
	AnnualIncrementPct   float64 // porcentaje de incremento anual (ej. 5.5 = 5.5 %)
	IsActive             bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
