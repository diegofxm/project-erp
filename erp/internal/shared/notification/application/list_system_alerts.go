// Package application contiene los casos de uso de notification.
package application

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/notification/domain"
)

const warnDays = 30

var monthsEs = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

func fmtDate(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthsEs[t.Month()-1], t.Year())
}

// daysUntil replica Math.floor((target-now)/86_400_000) del frontend original,
// incluida su semántica para diferencias negativas (certificado/rango ya vencido).
func daysUntil(t time.Time) int {
	return int(math.Floor(time.Until(t).Hours() / 24))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func docTypeLabel(code string) string {
	switch code {
	case "01":
		return "Factura"
	case "91":
		return "Nota Crédito"
	case "92":
		return "Nota Débito"
	default:
		return code
	}
}

func rangeLabel(r domain.RangeInfo) string {
	label := docTypeLabel(r.DianDocumentTypeCode)
	if r.Prefix != "" {
		return fmt.Sprintf("%q (%s)", r.Prefix, label)
	}
	return label
}

// ListSystemAlertsUseCase genera los avisos de sistema (certificado y rangos de
// numeración por vencer/vencidos/agotados) de una empresa. Misma regla de
// negocio y mismos textos que tenía frontend/src/context/NotificationContext.tsx
// (WARN_DAYS = 30) — movidos aquí para que cualquier consumidor futuro (otro
// canal, otro módulo) reutilice la misma lógica en vez de reimplementarla.
type ListSystemAlertsUseCase struct {
	company   domain.CompanyPort
	numbering domain.NumberingPort
}

func NewListSystemAlertsUseCase(company domain.CompanyPort, numbering domain.NumberingPort) *ListSystemAlertsUseCase {
	return &ListSystemAlertsUseCase{company: company, numbering: numbering}
}

func (uc *ListSystemAlertsUseCase) Execute(ctx context.Context, companyID uuid.UUID) ([]domain.Alert, error) {
	var out []domain.Alert

	cert, err := uc.company.GetCertificateInfo(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("consultar certificado: %w", err)
	}
	if cert.HasCertificate && cert.ExpiresAt != nil {
		days := daysUntil(*cert.ExpiresAt)
		exp := fmtDate(*cert.ExpiresAt)
		switch {
		case days < 0:
			out = append(out, domain.Alert{
				ID:       fmt.Sprintf("cert_expired:%s", cert.ExpiresAt.Format(time.RFC3339)),
				Severity: domain.SeverityDanger,
				Title:    "Certificado digital vencido",
				Message:  fmt.Sprintf("Venció el %s. No puedes firmar ni enviar documentos a la DIAN hasta renovarlo.", exp),
				LinkTo:   "/settings/company",
			})
		case days <= warnDays:
			out = append(out, domain.Alert{
				ID:       fmt.Sprintf("cert_expiring:%s", cert.ExpiresAt.Format(time.RFC3339)),
				Severity: domain.SeverityWarning,
				Title:    fmt.Sprintf("Certificado digital vence en %d día%s", days, plural(days)),
				Message:  fmt.Sprintf("Vence el %s. Renuévalo antes para no interrumpir la facturación.", exp),
				LinkTo:   "/settings/company",
			})
		}
	}

	ranges, err := uc.numbering.ListByCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("consultar rangos de numeración: %w", err)
	}
	for _, r := range ranges {
		label := rangeLabel(r)
		switch r.Status {
		case "expired":
			out = append(out, domain.Alert{
				ID:       fmt.Sprintf("range_expired:%s", r.ID),
				Severity: domain.SeverityDanger,
				Title:    fmt.Sprintf("Resolución %s vencida", label),
				Message:  fmt.Sprintf("Venció el %s. Sin resolución vigente no puedes numerar documentos nuevos.", fmtDate(r.ValidTo)),
				LinkTo:   "/settings/company",
			})
		case "active":
			days := daysUntil(r.ValidTo)
			if days >= 0 && days <= warnDays {
				out = append(out, domain.Alert{
					ID:       fmt.Sprintf("range_expiring:%s:%s", r.ID, r.ValidTo.Format(time.RFC3339)),
					Severity: domain.SeverityWarning,
					Title:    fmt.Sprintf("Resolución %s vence en %d día%s", label, days, plural(days)),
					Message:  fmt.Sprintf("Vigente hasta el %s. Tramita la renovación ante la DIAN con anticipación.", fmtDate(r.ValidTo)),
					LinkTo:   "/settings/company",
				})
			}
		case "exhausted":
			out = append(out, domain.Alert{
				ID:       fmt.Sprintf("range_exhausted:%s", r.ID),
				Severity: domain.SeverityDanger,
				Title:    fmt.Sprintf("Resolución %s agotada", label),
				Message:  "Se usaron todos los números autorizados. Registra una nueva resolución para seguir facturando.",
				LinkTo:   "/settings/company",
			})
		}
	}

	return out, nil
}
