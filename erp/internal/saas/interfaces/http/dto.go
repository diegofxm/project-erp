package http

import (
	"time"

	"github.com/diegofxm/erp/internal/saas/application"
	"github.com/diegofxm/erp/internal/saas/domain"
)

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatDate(*t)
}

type moduleDTO struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toModuleDTOs(list []domain.Module) []moduleDTO {
	out := make([]moduleDTO, len(list))
	for i, m := range list {
		out[i] = moduleDTO{Code: m.Code, Name: m.Name, Description: m.Description}
	}
	return out
}

type planDTO struct {
	ID                         string   `json:"id"`
	Code                       string   `json:"code"`
	Name                       string   `json:"name"`
	Description                string   `json:"description"`
	BillingCycle               string   `json:"billing_cycle"`
	PriceCents                 int64    `json:"price_cents"`
	IncludedDocuments          *int     `json:"included_documents"`
	PricePerExtraDocumentCents int64    `json:"price_per_extra_document_cents"`
	RequiresCertificate        bool     `json:"requires_certificate"`
	CertificatePriceCents      int64    `json:"certificate_price_cents"`
	AnnualIncrementPct         float64  `json:"annual_increment_pct"`
	IsInternal                 bool     `json:"is_internal"`
	IsActive                   bool     `json:"is_active"`
	Modules                    []string `json:"modules"`
	CreatedAt                  string   `json:"created_at"`
	UpdatedAt                  string   `json:"updated_at"`
}

func toPlanDTO(p *domain.Plan) planDTO {
	modules := p.ModuleCodes
	if modules == nil {
		modules = []string{}
	}
	return planDTO{
		ID: p.ID.String(), Code: p.Code, Name: p.Name, Description: p.Description,
		BillingCycle: string(p.BillingCycle), PriceCents: p.PriceCents,
		IncludedDocuments: p.IncludedDocuments, PricePerExtraDocumentCents: p.PricePerExtraDocumentCents,
		RequiresCertificate: p.RequiresCertificate, CertificatePriceCents: p.CertificatePriceCents,
		AnnualIncrementPct: p.AnnualIncrementPct, IsInternal: p.IsInternal, IsActive: p.IsActive,
		Modules: modules, CreatedAt: formatDate(p.CreatedAt), UpdatedAt: formatDate(p.UpdatedAt),
	}
}

func toPlanDTOs(list []domain.Plan) []planDTO {
	out := make([]planDTO, len(list))
	for i, p := range list {
		out[i] = toPlanDTO(&p)
	}
	return out
}

type settingsDTO struct {
	IVARateBP int    `json:"iva_rate_bp"`
	UpdatedAt string `json:"updated_at"`
}

func toSettingsDTO(s *domain.Settings) settingsDTO {
	return settingsDTO{IVARateBP: s.IVARateBP, UpdatedAt: formatDate(s.UpdatedAt)}
}

type subscriptionDTO struct {
	ID                   string `json:"id"`
	CompanyID            string `json:"company_id"`
	PlanID               string `json:"plan_id"`
	HasOwnCertificate    bool   `json:"has_own_certificate"`
	Status               string `json:"status"`
	ContractedPriceCents int64  `json:"contracted_price_cents"`
	CurrentPeriodStart   string `json:"current_period_start"`
	CurrentPeriodEnd     string `json:"current_period_end"`
	CertExpiresAt        string `json:"cert_expires_at,omitempty"`
}

func toSubscriptionDTO(s *domain.Subscription) subscriptionDTO {
	return subscriptionDTO{
		ID: s.ID.String(), CompanyID: s.CompanyID.String(), PlanID: s.PlanID.String(),
		HasOwnCertificate: s.HasOwnCertificate, Status: string(s.Status),
		ContractedPriceCents: s.ContractedPriceCents, CurrentPeriodStart: formatDate(s.CurrentPeriodStart),
		CurrentPeriodEnd: formatDate(s.CurrentPeriodEnd), CertExpiresAt: formatDatePtr(s.CertExpiresAt),
	}
}

type billingEntryDTO struct {
	CompanyID         string `json:"company_id"`
	BusinessName      string `json:"business_name"`
	NIT               string `json:"nit"`
	PlanName          string `json:"plan_name"`
	DocumentsIncluded *int   `json:"documents_included"`
	DocumentsUsed     int    `json:"documents_used"`
	OverageDocuments  int    `json:"overage_documents"`
	BaseCents         int64  `json:"base_cents"`
	OverageCents      int64  `json:"overage_cents"`
	IVACents          int64  `json:"iva_cents"`
	TotalCents        int64  `json:"total_cents"`
}

func toBillingEntryDTOs(list []application.BillingEntry) []billingEntryDTO {
	out := make([]billingEntryDTO, len(list))
	for i, e := range list {
		out[i] = billingEntryDTO{
			CompanyID: e.CompanyID.String(), BusinessName: e.BusinessName, NIT: e.NIT,
			PlanName: e.PlanName, DocumentsIncluded: e.DocumentsIncluded, DocumentsUsed: e.DocumentsUsed,
			OverageDocuments: e.OverageDocuments, BaseCents: e.BaseCents, OverageCents: e.OverageCents,
			IVACents: e.IVACents, TotalCents: e.TotalCents,
		}
	}
	return out
}

type renewalEntryDTO struct {
	CompanyID        string `json:"company_id"`
	BusinessName     string `json:"business_name"`
	NIT              string `json:"nit"`
	PlanName         string `json:"plan_name"`
	CurrentPeriodEnd string `json:"current_period_end"`
	DaysUntilRenewal int    `json:"days_until_renewal"`
	RenewalCents     int64  `json:"renewal_cents"`
}

func toRenewalEntryDTOs(list []application.RenewalEntry) []renewalEntryDTO {
	out := make([]renewalEntryDTO, len(list))
	for i, e := range list {
		out[i] = renewalEntryDTO{
			CompanyID: e.CompanyID.String(), BusinessName: e.BusinessName, NIT: e.NIT,
			PlanName: e.PlanName, CurrentPeriodEnd: e.CurrentPeriodEnd,
			DaysUntilRenewal: e.DaysUntilRenewal, RenewalCents: e.RenewalCents,
		}
	}
	return out
}

type paymentDTO struct {
	ID             string `json:"id"`
	CompanyID      string `json:"company_id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	Type           string `json:"type"`
	AmountCents    int64  `json:"amount_cents"`
	Note           string `json:"note"`
	PaidAt         string `json:"paid_at"`
}

func toPaymentDTO(p *domain.Payment) paymentDTO {
	dto := paymentDTO{
		ID: p.ID.String(), CompanyID: p.CompanyID.String(), Type: string(p.Type),
		AmountCents: p.AmountCents, Note: p.Note, PaidAt: formatDate(p.PaidAt),
	}
	if p.SubscriptionID != nil {
		dto.SubscriptionID = p.SubscriptionID.String()
	}
	return dto
}

func toPaymentDTOs(list []domain.Payment) []paymentDTO {
	out := make([]paymentDTO, len(list))
	for i, p := range list {
		out[i] = toPaymentDTO(&p)
	}
	return out
}

type prospectDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	NIT        string `json:"nit"`
	HasCedula  bool   `json:"has_cedula"`
	HasRUT     bool   `json:"has_rut"`
	Status     string `json:"status"`
	Notes      string `json:"notes,omitempty"`
	ReviewedAt string `json:"reviewed_at,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func toProspectDTO(p *domain.Prospect) prospectDTO {
	return prospectDTO{
		ID: p.ID.String(), Name: p.Name, Email: p.Email, NIT: p.NIT,
		HasCedula: len(p.CedulaFile) > 0, HasRUT: len(p.RUTFile) > 0, Status: string(p.Status),
		Notes: p.Notes, ReviewedAt: formatDatePtr(p.ReviewedAt), CreatedAt: formatDate(p.CreatedAt),
	}
}

func toProspectDTOs(list []domain.Prospect) []prospectDTO {
	out := make([]prospectDTO, len(list))
	for i, p := range list {
		out[i] = toProspectDTO(&p)
	}
	return out
}

type platformUserDTO struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	Name             string `json:"name"`
	Role             string `json:"role"`
	IsSuperAdmin     bool   `json:"is_superadmin"`
	IsActive         bool   `json:"is_active"`
	InviteAcceptedAt string `json:"invite_accepted_at,omitempty"`
	CreatedAt        string `json:"created_at"`
}

func toPlatformUserDTOs(list []domain.PlatformUser) []platformUserDTO {
	out := make([]platformUserDTO, len(list))
	for i, u := range list {
		out[i] = platformUserDTO{
			ID: u.ID.String(), Email: u.Email, Name: u.Name, Role: u.Role,
			IsSuperAdmin: u.IsSuperAdmin, IsActive: u.IsActive,
			InviteAcceptedAt: formatDatePtr(u.InviteAcceptedAt), CreatedAt: formatDate(u.CreatedAt),
		}
	}
	return out
}

type myPlanDTO struct {
	PlanName          string   `json:"plan_name"`
	Modules           []string `json:"modules"`
	IncludedDocuments *int     `json:"included_documents"`
	DocumentsUsed     int      `json:"documents_used"`
	CurrentPeriodEnd  string   `json:"current_period_end"`
	ContractedCents   int64    `json:"contracted_cents"`
	HasOwnCertificate bool     `json:"has_own_certificate"`
	CertExpiresAt     string   `json:"cert_expires_at,omitempty"`
}

func toMyPlanDTO(p *application.MyPlan) myPlanDTO {
	modules := p.ModuleCodes
	if modules == nil {
		modules = []string{}
	}
	return myPlanDTO{
		PlanName: p.PlanName, Modules: modules, IncludedDocuments: p.IncludedDocuments,
		DocumentsUsed: p.DocumentsUsed, CurrentPeriodEnd: p.CurrentPeriodEnd,
		ContractedCents: p.ContractedCents, HasOwnCertificate: p.HasOwnCertificate,
		CertExpiresAt: p.CertExpiresAt,
	}
}
