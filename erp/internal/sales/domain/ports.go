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

// Customer es la vista de thirdparty.Party que necesita sales — para el chequeo de cupo de
// cartera (checkCredit) y para representar el cliente en el PDF/email de la cotización. No
// necesita los campos tributarios DIAN completos (eso lo usa electronic, no sales).
type Customer struct {
	Name                   string
	IdentificationTypeCode string
	IdentificationNumber   string
	CheckDigit             string
	AddressLine            string
	Phone                  string
	Email                  string
	CreditLimit            *float64
}

// CustomerPort lee el cliente de una venta/cotización.
type CustomerPort interface {
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Customer, error)
}
