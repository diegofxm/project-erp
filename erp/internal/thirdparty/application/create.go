package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/shared/nit"
	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

type CreateUseCase struct {
	repo domain.Repository
}

func NewCreateUseCase(repo domain.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

type SaveRequest struct {
	IdentificationTypeCode     string   `json:"identification_type_code"`
	IdentificationNumber       string   `json:"identification_number"`
	CheckDigit                 string   `json:"check_digit"`
	EntityTypeCode             string   `json:"entity_type_code"`
	MerchantRegistrationNumber string   `json:"merchant_registration_number"`
	Name                       string   `json:"name"`
	TaxSchemeCode              string   `json:"tax_scheme_code"`
	TaxSchemeName              string   `json:"tax_scheme_name"`
	TaxRegimeCode              *string  `json:"tax_regime_code"`
	LiabilityCodes             []string `json:"liability_codes"`
	DepartmentCode             string   `json:"department_code"`
	MunicipalityCode           string   `json:"municipality_code"`
	AddressLine                string   `json:"address_line"`
	AddressCityName            string   `json:"address_city_name"`
	AddressStateName           string   `json:"address_state_name"`
	AddressCountryCode         string   `json:"address_country_code"`
	AddressCountryName         string   `json:"address_country_name"`
	Email                      string   `json:"email"`
	Phone                      string   `json:"phone"`
	CreditLimit                *float64 `json:"credit_limit"`
	PaymentTermsDays           int      `json:"payment_terms_days"`
}

// Execute crea un tercero nuevo con el rol pedido. Si ya existe otro tercero con la misma
// identificación pero sin ese rol (ej. ya es Proveedor y se está dando de alta como Cliente), se
// le agrega el rol al registro existente en vez de duplicarlo — esto es lo que reemplaza la idea
// original de "avisar y copiar datos": ya no hace falta avisar, el alta encuentra al tercero
// existente y lo completa directamente.
func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, role domain.Role, req SaveRequest) (*domain.Party, error) {
	checkDigit := resolveCheckDigit(req)

	existing, err := uc.repo.GetByIdentification(ctx, companyID, req.IdentificationTypeCode, req.IdentificationNumber)
	if err == nil {
		if role == domain.RoleCustomer && existing.IsCustomer {
			return nil, domain.ErrDuplicateCustomer
		}
		if role == domain.RoleSupplier && existing.IsSupplier {
			return nil, domain.ErrDuplicateSupplier
		}
		applyShared(existing, req, checkDigit)
		applyRole(existing, role, req)
		return uc.repo.Update(ctx, *existing)
	}
	if !errors.Is(err, domain.ErrPartyNotFound) {
		return nil, err
	}

	p := domain.Party{ID: uuid.New(), CompanyID: companyID, IsActive: true}
	applyShared(&p, req, checkDigit)
	applyRole(&p, role, req)
	return uc.repo.Save(ctx, p)
}

func resolveCheckDigit(req SaveRequest) string {
	checkDigit := req.CheckDigit
	if req.IdentificationTypeCode == "31" {
		if dv, err := nit.ComputeCheckDigit(req.IdentificationNumber); err == nil {
			checkDigit = dv
		}
	}
	return checkDigit
}

func applyShared(p *domain.Party, req SaveRequest, checkDigit string) {
	p.IdentificationTypeCode = req.IdentificationTypeCode
	p.IdentificationNumber = req.IdentificationNumber
	p.CheckDigit = checkDigit
	p.EntityTypeCode = req.EntityTypeCode
	p.MerchantRegistrationNumber = req.MerchantRegistrationNumber
	p.Name = req.Name
	p.TaxSchemeCode = req.TaxSchemeCode
	p.TaxSchemeName = req.TaxSchemeName
	p.TaxRegimeCode = req.TaxRegimeCode
	p.LiabilityCodes = req.LiabilityCodes
	if p.LiabilityCodes == nil {
		p.LiabilityCodes = []string{}
	}
	p.DepartmentCode = req.DepartmentCode
	p.MunicipalityCode = req.MunicipalityCode
	p.AddressLine = req.AddressLine
	p.AddressCityName = req.AddressCityName
	p.AddressStateName = req.AddressStateName
	p.AddressCountryCode = req.AddressCountryCode
	p.AddressCountryName = req.AddressCountryName
	p.Email = req.Email
	p.Phone = req.Phone
}

func applyRole(p *domain.Party, role domain.Role, req SaveRequest) {
	switch role {
	case domain.RoleCustomer:
		p.IsCustomer = true
		p.CreditLimit = req.CreditLimit
	case domain.RoleSupplier:
		p.IsSupplier = true
		p.PaymentTermsDays = req.PaymentTermsDays
	}
}
