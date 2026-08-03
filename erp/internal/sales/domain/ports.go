package domain

import (
	"context"

	"github.com/google/uuid"
)

// CompanyInfo es la vista de Company que necesita sales para representar gráficamente una
// cotización (PDF/email) — no requiere nada específico de DIAN, a diferencia de
// electronic.CompanyInfo.
type CompanyInfo struct {
	BusinessName    string
	TradeName       string
	NIT             string
	CheckDigit      string
	AddressLine     string
	MunicipalityCode string
	Phone           string
	Email           string
	Logo            []byte
	LogoContentType string
}

// CompanyPort lee la empresa emisora de la cotización.
type CompanyPort interface {
	GetCompany(ctx context.Context, id uuid.UUID) (*CompanyInfo, error)
}
