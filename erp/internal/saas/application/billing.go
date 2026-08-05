package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

// daysUntil redondea hacia arriba para que "vence en menos de 1 día" muestre 1, no 0 — más
// intuitivo para el badge de urgencia del panel de renovaciones. Negativo si ya venció.
func daysUntil(t time.Time) int {
	d := time.Until(t)
	if d <= 0 {
		return -int(-d / (24 * time.Hour))
	}
	return int((d + 24*time.Hour - time.Nanosecond) / (24 * time.Hour))
}

// BillingEntry es el resumen de facturación de una empresa en su período vigente — usado por el
// panel superadmin (GET /admin/billing/summary).
type BillingEntry struct {
	CompanyID         uuid.UUID
	BusinessName      string
	NIT               string
	PlanName          string
	DocumentsIncluded *int // nil = ilimitado
	DocumentsUsed     int
	OverageDocuments  int
	BaseCents         int64
	OverageCents      int64
	IVACents          int64
	TotalCents        int64
}

// RenewalEntry es una suscripción próxima a vencer (o ya vencida) — GET /admin/billing/renewals.
type RenewalEntry struct {
	CompanyID        uuid.UUID
	BusinessName     string
	NIT              string
	PlanName         string
	CurrentPeriodEnd string // YYYY-MM-DD, ya formateado para no acoplar el frontend a time.Time
	DaysUntilRenewal int    // negativo = ya vencida
	RenewalCents     int64
}

type BillingUseCase struct {
	subs     domain.SubscriptionRepository
	plans    domain.PlanRepository
	settings domain.SettingsRepository
	docs     domain.DocumentCounterPort
	company  domain.CompanyPort
}

func NewBillingUseCase(
	subs domain.SubscriptionRepository,
	plans domain.PlanRepository,
	settings domain.SettingsRepository,
	docs domain.DocumentCounterPort,
	company domain.CompanyPort,
) *BillingUseCase {
	return &BillingUseCase{subs: subs, plans: plans, settings: settings, docs: docs, company: company}
}

func applyIVA(baseCents int64, ivaRateBP int) int64 {
	return baseCents * int64(ivaRateBP) / 10000
}

func (uc *BillingUseCase) Summary(ctx context.Context) ([]BillingEntry, error) {
	subs, err := uc.subs.ListAllActive(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := uc.settings.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtener configuración de IVA: %w", err)
	}

	out := make([]BillingEntry, 0, len(subs))
	for _, sub := range subs {
		plan, err := uc.plans.GetByID(ctx, sub.PlanID)
		if err != nil {
			continue // plan borrado/inconsistente — no debería pasar, se omite en vez de romper el reporte completo
		}
		company, err := uc.company.GetCompany(ctx, sub.CompanyID)
		if err != nil {
			continue
		}
		used, err := uc.docs.CountInPeriod(ctx, sub.CompanyID, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
		if err != nil {
			return nil, fmt.Errorf("contar documentos de %s: %w", company.BusinessName, err)
		}

		var overageDocs int
		var overageCents int64
		if plan.IncludedDocuments != nil && used > *plan.IncludedDocuments {
			overageDocs = used - *plan.IncludedDocuments
			overageCents = int64(overageDocs) * plan.PricePerExtraDocumentCents
		}

		base := sub.ContractedPriceCents + overageCents
		iva := applyIVA(base, settings.IVARateBP)

		out = append(out, BillingEntry{
			CompanyID: sub.CompanyID, BusinessName: company.BusinessName, NIT: company.NIT,
			PlanName: plan.Name, DocumentsIncluded: plan.IncludedDocuments, DocumentsUsed: used,
			OverageDocuments: overageDocs, BaseCents: sub.ContractedPriceCents, OverageCents: overageCents,
			IVACents: iva, TotalCents: base + iva,
		})
	}
	return out, nil
}

func (uc *BillingUseCase) Renewals(ctx context.Context, withinDays int) ([]RenewalEntry, error) {
	subs, err := uc.subs.ListUpcomingRenewals(ctx, withinDays)
	if err != nil {
		return nil, err
	}

	out := make([]RenewalEntry, 0, len(subs))
	for _, sub := range subs {
		plan, err := uc.plans.GetByID(ctx, sub.PlanID)
		if err != nil {
			continue
		}
		company, err := uc.company.GetCompany(ctx, sub.CompanyID)
		if err != nil {
			continue
		}
		out = append(out, RenewalEntry{
			CompanyID: sub.CompanyID, BusinessName: company.BusinessName, NIT: company.NIT,
			PlanName: plan.Name, CurrentPeriodEnd: sub.CurrentPeriodEnd.Format("2006-01-02"),
			DaysUntilRenewal: daysUntil(sub.CurrentPeriodEnd), RenewalCents: sub.ContractedPriceCents,
		})
	}
	return out, nil
}
