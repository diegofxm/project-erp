package domain

import (
	"context"

	"github.com/google/uuid"
)

// BillingStats agrega métricas de facturación electrónica para el dashboard de stats/ -- vive en
// electronic (dueño de la tabla documents) en vez de que stats/ consulte electronic.documents
// directamente por SQL (ver auditoría 2026-08-09, Fase 2 punto 15). stats/infrastructure/electronic
// mapea esto a su propio domain.BillingStats.
type BillingStats struct {
	CurrentMonth  PeriodStats
	PreviousMonth PeriodStats
	YTD           PeriodStats
	ByType        []TypeStats
	Series        []MonthSeries
}

type PeriodStats struct {
	RevenueCents  int64
	DocumentCount int
	AcceptedCount int
	RejectedCount int
	DraftCount    int
}

type TypeStats struct {
	TypeCode     string
	TypeName     string
	Count        int
	RevenueCents int64
}

type MonthSeries struct {
	Month         string // "YYYY-MM"
	RevenueCents  int64
	Count         int
	AcceptedCount int
}

// BillingStatsRepository expone las agregaciones sobre electronic.documents que stats/ necesita
// para su dashboard.
type BillingStatsRepository interface {
	GetBillingStats(ctx context.Context, companyID uuid.UUID) (*BillingStats, error)
}
