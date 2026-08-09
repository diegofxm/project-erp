package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/stats/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetBillingStats ejecuta tres queries SQL sobre electronic.documents para construir
// las métricas del dashboard. Filtra por company_id y usa la zona horaria de Bogotá.
func (r *Repository) GetBillingStats(ctx context.Context, companyID uuid.UUID) (*domain.BillingStats, error) {
	stats := &domain.BillingStats{}

	// ── 1. Stats por período: mes actual, mes anterior, acumulado anual ──────────────────
	err := r.pool.QueryRow(ctx, `
		SELECT
			-- Mes actual
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status IN ('rejected', 'send_error', 'send_unknown')),
			-- Borradores: sin filtro de fecha (pendientes ahora)
			COUNT(*) FILTER (WHERE status = 'draft'),
			-- Mes anterior
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status IN ('rejected', 'send_error', 'send_unknown')),
			-- Acumulado año
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted')
		FROM electronic.documents
		WHERE company_id = $1`,
		companyID,
	).Scan(
		&stats.CurrentMonth.RevenueCents,
		&stats.CurrentMonth.DocumentCount,
		&stats.CurrentMonth.AcceptedCount,
		&stats.CurrentMonth.RejectedCount,
		&stats.CurrentMonth.DraftCount,
		&stats.PreviousMonth.RevenueCents,
		&stats.PreviousMonth.DocumentCount,
		&stats.PreviousMonth.AcceptedCount,
		&stats.PreviousMonth.RejectedCount,
		&stats.YTD.RevenueCents,
		&stats.YTD.DocumentCount,
		&stats.YTD.AcceptedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats periods: %w", err)
	}

	// ── 2. Desglose por tipo de documento (mes actual) ───────────────────────────────────
	typeRows, err := r.pool.Query(ctx, `
		SELECT
			dian_document_type_code,
			COUNT(*) FILTER (WHERE status != 'draft'),
			COALESCE(SUM(totals_payable_cents) FILTER (WHERE status = 'accepted'), 0)
		FROM electronic.documents
		WHERE company_id = $1
		  AND issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
		GROUP BY dian_document_type_code
		ORDER BY COUNT(*) FILTER (WHERE status != 'draft') DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats by_type: %w", err)
	}
	defer typeRows.Close()

	typeNames := map[string]string{
		"01": "Factura Electrónica",
		"91": "Nota Crédito",
		"92": "Nota Débito",
		"05": "Documento Soporte",
	}
	for typeRows.Next() {
		var ts domain.TypeStats
		if err := typeRows.Scan(&ts.TypeCode, &ts.Count, &ts.RevenueCents); err != nil {
			return nil, fmt.Errorf("billing stats by_type scan: %w", err)
		}
		if name, ok := typeNames[ts.TypeCode]; ok {
			ts.TypeName = name
		} else {
			ts.TypeName = ts.TypeCode
		}
		stats.ByType = append(stats.ByType, ts)
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("billing stats by_type rows: %w", err)
	}
	if stats.ByType == nil {
		stats.ByType = []domain.TypeStats{}
	}

	// ── 3. Serie mensual: últimos 12 meses ──────────────────────────────────────────────
	seriesRows, err := r.pool.Query(ctx, `
		SELECT
			to_char(date_trunc('month', issue_date), 'YYYY-MM'),
			COUNT(*) FILTER (WHERE status != 'draft'),
			COUNT(*) FILTER (WHERE status = 'accepted'),
			COALESCE(SUM(totals_payable_cents) FILTER (WHERE status = 'accepted'), 0)
		FROM electronic.documents
		WHERE company_id = $1
		  AND issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '11 months')::date
		  AND status != 'draft'
		GROUP BY date_trunc('month', issue_date)
		ORDER BY date_trunc('month', issue_date)`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats series: %w", err)
	}
	defer seriesRows.Close()

	for seriesRows.Next() {
		var ms domain.MonthSeries
		if err := seriesRows.Scan(&ms.Month, &ms.Count, &ms.AcceptedCount, &ms.RevenueCents); err != nil {
			return nil, fmt.Errorf("billing stats series scan: %w", err)
		}
		stats.Series = append(stats.Series, ms)
	}
	if err := seriesRows.Err(); err != nil {
		return nil, fmt.Errorf("billing stats series rows: %w", err)
	}
	if stats.Series == nil {
		stats.Series = []domain.MonthSeries{}
	}

	return stats, nil
}
