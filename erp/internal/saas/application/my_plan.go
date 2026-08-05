package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

// MyPlan es la foto del plan contratado por la empresa activa — alimenta la página "Mi plan" del
// usuario normal y el filtro de módulos del sidebar (frontend/src/components/Sidebar.tsx).
type MyPlan struct {
	PlanName          string
	ModuleCodes       []string
	IncludedDocuments *int // nil = ilimitado
	DocumentsUsed     int
	CurrentPeriodEnd  string // YYYY-MM-DD
	ContractedCents   int64
	HasOwnCertificate bool
	CertExpiresAt     string // YYYY-MM-DD, vacío si no aplica
}

type MyPlanUseCase struct {
	subs  domain.SubscriptionRepository
	plans domain.PlanRepository
	docs  domain.DocumentCounterPort
}

func NewMyPlanUseCase(subs domain.SubscriptionRepository, plans domain.PlanRepository, docs domain.DocumentCounterPort) *MyPlanUseCase {
	return &MyPlanUseCase{subs: subs, plans: plans, docs: docs}
}

func (uc *MyPlanUseCase) Get(ctx context.Context, companyID uuid.UUID) (*MyPlan, error) {
	sub, err := uc.subs.GetActive(ctx, companyID)
	if err != nil {
		return nil, err
	}
	plan, err := uc.plans.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	used, err := uc.docs.CountInPeriod(ctx, companyID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
	if err != nil {
		return nil, err
	}

	out := &MyPlan{
		PlanName: plan.Name, ModuleCodes: plan.ModuleCodes, IncludedDocuments: plan.IncludedDocuments,
		DocumentsUsed: used, CurrentPeriodEnd: sub.CurrentPeriodEnd.Format("2006-01-02"),
		ContractedCents: sub.ContractedPriceCents, HasOwnCertificate: sub.HasOwnCertificate,
	}
	if sub.CertExpiresAt != nil {
		out.CertExpiresAt = sub.CertExpiresAt.Format("2006-01-02")
	}
	return out, nil
}
