package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/supplier/domain"
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
	PaymentTermsDays           int      `json:"payment_terms_days"`
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req SaveRequest) (*domain.Supplier, error) {
	_, err := uc.repo.GetByIdentification(ctx, companyID, req.IdentificationTypeCode, req.IdentificationNumber)
	if err == nil {
		return nil, domain.ErrDuplicateSupplier
	}

	s := domain.Supplier{
		ID:                         uuid.New(),
		CompanyID:                  companyID,
		IdentificationTypeCode:     req.IdentificationTypeCode,
		IdentificationNumber:       req.IdentificationNumber,
		CheckDigit:                 req.CheckDigit,
		EntityTypeCode:             req.EntityTypeCode,
		MerchantRegistrationNumber: req.MerchantRegistrationNumber,
		Name:                       req.Name,
		TaxSchemeCode:              req.TaxSchemeCode,
		TaxSchemeName:              req.TaxSchemeName,
		TaxRegimeCode:              req.TaxRegimeCode,
		LiabilityCodes:             req.LiabilityCodes,
		DepartmentCode:             req.DepartmentCode,
		MunicipalityCode:           req.MunicipalityCode,
		AddressLine:                req.AddressLine,
		AddressCityName:            req.AddressCityName,
		AddressStateName:           req.AddressStateName,
		AddressCountryCode:         req.AddressCountryCode,
		AddressCountryName:         req.AddressCountryName,
		Email:                      req.Email,
		Phone:                      req.Phone,
		PaymentTermsDays:           req.PaymentTermsDays,
		IsActive:                   true,
	}

	return uc.repo.Save(ctx, s)
}
