// Package saas implementa company/application.PlanAssigner envolviendo el módulo saas — mismo
// patrón de puerto local que ya usan sales/infrastructure/company, saas/infrastructure/company,
// etc. Evita que company/ importe saas/ directamente.
package saas

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	saasapp "github.com/diegofxm/erp/internal/saas/application"
	saasdomain "github.com/diegofxm/erp/internal/saas/domain"
)

type Adapter struct {
	subscriptions *saasapp.SubscriptionUseCase
	plans         saasdomain.PlanRepository
}

func New(subscriptions *saasapp.SubscriptionUseCase, plans saasdomain.PlanRepository) *Adapter {
	return &Adapter{subscriptions: subscriptions, plans: plans}
}

// AssignInternalPlan busca el plan sembrado con code="interno" (ver saas/infrastructure/persistence/postgres/seed)
// y lo asigna como suscripción activa de la empresa — sin certificado propio, ese plan no lo exige.
func (a *Adapter) AssignInternalPlan(ctx context.Context, companyID uuid.UUID) error {
	list, err := a.plans.List(ctx)
	if err != nil {
		return fmt.Errorf("listar planes: %w", err)
	}
	var internalID uuid.UUID
	for _, p := range list {
		if p.Code == "interno" {
			internalID = p.ID
			break
		}
	}
	if internalID == uuid.Nil {
		return fmt.Errorf("plan interno no encontrado en el catálogo — ¿se corrió el seed de saas?")
	}
	_, err = a.subscriptions.Assign(ctx, companyID, internalID, false)
	return err
}
