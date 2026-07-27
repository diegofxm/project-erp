package documents

import (
	"context"

	"github.com/google/uuid"
)

// BillingStats agrega métricas de facturación para el dashboard.
// Solo cuenta documentos con issue_date no nulo (excluye borradores),
// y el revenue_cents solo incluye los de status=accepted (autorizados por la DIAN).
type BillingStats struct {
	CurrentMonth  PeriodStats   `json:"current_month"`
	PreviousMonth PeriodStats   `json:"previous_month"`
	YTD           PeriodStats   `json:"ytd"`
	ByType        []TypeStats   `json:"by_type"`
	Series        []MonthSeries `json:"series"`
}

// PeriodStats resume los documentos y montos de un período.
type PeriodStats struct {
	RevenueCents  int64 `json:"revenue_cents"`
	DocumentCount int   `json:"document_count"`
	AcceptedCount int   `json:"accepted_count"`
	RejectedCount int   `json:"rejected_count"`
	DraftCount    int   `json:"draft_count"`
}

// TypeStats desglosa por tipo de documento DIAN (01/91/92/05).
type TypeStats struct {
	TypeCode     string `json:"type_code"`
	TypeName     string `json:"type_name"`
	Count        int    `json:"count"`
	RevenueCents int64  `json:"revenue_cents"`
}

// MonthSeries es un punto de la serie temporal mensual (últimos 12 meses).
type MonthSeries struct {
	Month         string `json:"month"` // "2025-08"
	RevenueCents  int64  `json:"revenue_cents"`
	Count         int    `json:"count"`
	AcceptedCount int    `json:"accepted_count"`
}

// GetBillingStats delega al repositorio.
func (s *Service) GetBillingStats(ctx context.Context, issuerID uuid.UUID) (*BillingStats, error) {
	return s.repo.GetBillingStats(ctx, issuerID)
}
