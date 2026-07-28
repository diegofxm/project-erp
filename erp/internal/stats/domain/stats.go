package domain

// BillingStats agrega métricas de facturación electrónica para el dashboard.
type BillingStats struct {
	CurrentMonth  PeriodStats   `json:"current_month"`
	PreviousMonth PeriodStats   `json:"previous_month"`
	YTD           PeriodStats   `json:"ytd"`
	ByType        []TypeStats   `json:"by_type"`
	Series        []MonthSeries `json:"series"`
}

// PeriodStats resume documentos y montos de un período.
type PeriodStats struct {
	RevenueCents  int64 `json:"revenue_cents"`
	DocumentCount int   `json:"document_count"`
	AcceptedCount int   `json:"accepted_count"`
	RejectedCount int   `json:"rejected_count"`
	DraftCount    int   `json:"draft_count"`
}

// TypeStats desglosa por tipo DIAN (01/91/92/05).
type TypeStats struct {
	TypeCode     string `json:"type_code"`
	TypeName     string `json:"type_name"`
	Count        int    `json:"count"`
	RevenueCents int64  `json:"revenue_cents"`
}

// MonthSeries es un punto de la serie temporal mensual (últimos 12 meses).
type MonthSeries struct {
	Month         string `json:"month"` // "YYYY-MM"
	RevenueCents  int64  `json:"revenue_cents"`
	Count         int    `json:"count"`
	AcceptedCount int    `json:"accepted_count"`
}
