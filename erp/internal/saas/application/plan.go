package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type PlanUseCase struct {
	plans   domain.PlanRepository
	modules domain.ModuleRepository
}

func NewPlanUseCase(plans domain.PlanRepository, modules domain.ModuleRepository) *PlanUseCase {
	return &PlanUseCase{plans: plans, modules: modules}
}

func (uc *PlanUseCase) List(ctx context.Context) ([]domain.Plan, error) {
	return uc.plans.List(ctx)
}

func (uc *PlanUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	return uc.plans.GetByID(ctx, id)
}

func (uc *PlanUseCase) ListModules(ctx context.Context) ([]domain.Module, error) {
	return uc.modules.List(ctx)
}

// validateModules confirma que cada código de módulo exista en el catálogo — evita que un plan
// quede apuntando a un módulo mal escrito que nunca desbloquearía nada en el frontend.
func (uc *PlanUseCase) validateModules(ctx context.Context, codes []string) error {
	for _, code := range codes {
		if _, err := uc.modules.GetByCode(ctx, code); err != nil {
			return fmt.Errorf("módulo %q inválido: %w", code, err)
		}
	}
	return nil
}

func (uc *PlanUseCase) Create(ctx context.Context, p domain.Plan) (*domain.Plan, error) {
	if err := uc.validateModules(ctx, p.ModuleCodes); err != nil {
		return nil, err
	}
	return uc.plans.Create(ctx, p)
}

func (uc *PlanUseCase) Update(ctx context.Context, p domain.Plan) (*domain.Plan, error) {
	if err := uc.validateModules(ctx, p.ModuleCodes); err != nil {
		return nil, err
	}
	return uc.plans.Update(ctx, p)
}

// ApplyIncrement sube PriceCents en AnnualIncrementPct% — no es retroactivo: las suscripciones ya
// vigentes conservan su ContractedPriceCents hasta su propia renovación (ver SubscriptionUseCase.Renew).
func (uc *PlanUseCase) ApplyIncrement(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	p, err := uc.plans.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.AnnualIncrementPct <= 0 {
		return p, nil
	}
	p.PriceCents = p.PriceCents + int64(float64(p.PriceCents)*p.AnnualIncrementPct/100)
	return uc.plans.Update(ctx, *p)
}
