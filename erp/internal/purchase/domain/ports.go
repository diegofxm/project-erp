package domain

import (
	"context"

	"github.com/google/uuid"
)

// CompanyInfo es la vista de Company que necesita purchase para representar gráficamente una
// orden de compra (PDF/email) — mismo criterio que sales.CompanyInfo, no requiere nada
// específico de DIAN.
type CompanyInfo struct {
	BusinessName     string
	TradeName        string
	NIT              string
	CheckDigit       string
	AddressLine      string
	MunicipalityCode string
	Phone            string
	Email            string
	Logo             []byte
	LogoContentType  string
}

// CompanyPort lee la empresa compradora (emisora de la orden de compra).
type CompanyPort interface {
	GetCompany(ctx context.Context, id uuid.UUID) (*CompanyInfo, error)
}
