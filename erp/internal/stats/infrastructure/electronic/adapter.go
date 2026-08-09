// Package electronic implementa stats/domain.Repository envolviendo
// electronic/domain.BillingStatsRepository -- mismo patrón de puerto local que ya usan
// sales/infrastructure/thirdparty, electronic/infrastructure/company, etc. Antes,
// stats/infrastructure/persistence/postgres ejecutaba SQL directo contra electronic.documents
// desde otro módulo (ver auditoría 2026-08-09, Fase 2 punto 15); ahora esas queries viven en
// electronic (dueño real del schema) y acá solo se traduce su resultado al domain.BillingStats
// propio de stats.
package electronic

import (
	"context"

	"github.com/google/uuid"

	electronicdomain "github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/stats/domain"
)

type Adapter struct {
	repo electronicdomain.BillingStatsRepository
}

func New(repo electronicdomain.BillingStatsRepository) *Adapter {
	return &Adapter{repo: repo}
}

func (a *Adapter) GetBillingStats(ctx context.Context, companyID uuid.UUID) (*domain.BillingStats, error) {
	src, err := a.repo.GetBillingStats(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return &domain.BillingStats{
		CurrentMonth:  toPeriodStats(src.CurrentMonth),
		PreviousMonth: toPeriodStats(src.PreviousMonth),
		YTD:           toPeriodStats(src.YTD),
		ByType:        toTypeStats(src.ByType),
		Series:        toMonthSeries(src.Series),
	}, nil
}

func toPeriodStats(p electronicdomain.PeriodStats) domain.PeriodStats {
	return domain.PeriodStats{
		RevenueCents:  p.RevenueCents,
		DocumentCount: p.DocumentCount,
		AcceptedCount: p.AcceptedCount,
		RejectedCount: p.RejectedCount,
		DraftCount:    p.DraftCount,
	}
}

func toTypeStats(in []electronicdomain.TypeStats) []domain.TypeStats {
	out := make([]domain.TypeStats, len(in))
	for i, t := range in {
		out[i] = domain.TypeStats{
			TypeCode:     t.TypeCode,
			TypeName:     t.TypeName,
			Count:        t.Count,
			RevenueCents: t.RevenueCents,
		}
	}
	return out
}

func toMonthSeries(in []electronicdomain.MonthSeries) []domain.MonthSeries {
	out := make([]domain.MonthSeries, len(in))
	for i, m := range in {
		out[i] = domain.MonthSeries{
			Month:         m.Month,
			RevenueCents:  m.RevenueCents,
			Count:         m.Count,
			AcceptedCount: m.AcceptedCount,
		}
	}
	return out
}
