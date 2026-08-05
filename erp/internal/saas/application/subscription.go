package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type SubscriptionUseCase struct {
	subs  domain.SubscriptionRepository
	plans domain.PlanRepository
}

func NewSubscriptionUseCase(subs domain.SubscriptionRepository, plans domain.PlanRepository) *SubscriptionUseCase {
	return &SubscriptionUseCase{subs: subs, plans: plans}
}

func (uc *SubscriptionUseCase) Get(ctx context.Context, companyID uuid.UUID) (*domain.Subscription, error) {
	return uc.subs.GetActive(ctx, companyID)
}

// periodLength — cuánto dura un ciclo de facturación, usado tanto para el período de cobro como
// para el reseteo del cupo de documentos (el cupo resetea junto con el ciclo del plan).
func periodLength(cycle domain.BillingCycle) (years, months int) {
	if cycle == domain.BillingAnnual {
		return 1, 0
	}
	return 0, 1 // monthly y none comparten el mismo cupo mensual — none es $0, no tiene cobro
}

// Assign contrata (o cambia) el plan de una empresa — cancela la suscripción activa anterior si
// existía y crea una nueva. hasOwnCertificate solo importa si el plan exige certificado.
func (uc *SubscriptionUseCase) Assign(ctx context.Context, companyID, planID uuid.UUID, hasOwnCertificate bool) (*domain.Subscription, error) {
	plan, err := uc.plans.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !plan.IsActive {
		return nil, domain.ErrPlanInactive
	}

	if err := uc.subs.Cancel(ctx, companyID); err != nil {
		return nil, fmt.Errorf("cancelar suscripción anterior: %w", err)
	}

	now := time.Now()
	years, months := periodLength(plan.BillingCycle)
	periodEnd := now.AddDate(years, months, 0)

	contractedPrice := plan.PriceCents
	if plan.RequiresCertificate && !hasOwnCertificate {
		contractedPrice += plan.CertificatePriceCents
	}

	var certExpiresAt *time.Time
	if plan.RequiresCertificate && !hasOwnCertificate {
		t := now.AddDate(1, 0, 0)
		certExpiresAt = &t
	}

	return uc.subs.Create(ctx, domain.Subscription{
		CompanyID:            companyID,
		PlanID:               planID,
		HasOwnCertificate:    hasOwnCertificate,
		Status:               domain.SubscriptionActive,
		ContractedPriceCents: contractedPrice,
		CurrentPeriodStart:   now,
		CurrentPeriodEnd:     periodEnd,
		CertExpiresAt:        certExpiresAt,
	})
}

// Renew extiende el período vigente con el precio ACTUAL del plan (recoge cualquier incremento
// anual aplicado desde la última renovación — ver PlanUseCase.ApplyIncrement, no retroactivo a
// mitad de ciclo, pero sí se aplica en cada renovación).
func (uc *SubscriptionUseCase) Renew(ctx context.Context, companyID uuid.UUID) (*domain.Subscription, error) {
	sub, err := uc.subs.GetActive(ctx, companyID)
	if err != nil {
		return nil, err
	}
	plan, err := uc.plans.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}

	years, months := periodLength(plan.BillingCycle)
	start := sub.CurrentPeriodEnd
	if start.Before(time.Now()) {
		start = time.Now()
	}
	newEnd := start.AddDate(years, months, 0)

	contractedPrice := plan.PriceCents
	if plan.RequiresCertificate && !sub.HasOwnCertificate {
		contractedPrice += plan.CertificatePriceCents
	}

	return uc.subs.Renew(ctx, sub.ID, start, newEnd, contractedPrice)
}
