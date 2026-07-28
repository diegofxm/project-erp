package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

type UpdateProfileUseCase struct {
	repo domain.Repository
}

func NewUpdateProfileUseCase(repo domain.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

type UpdateProfileRequest struct {
	BusinessName                string             `json:"business_name"`
	TradeName                   string             `json:"trade_name"`
	IdentificationTypeCode      string             `json:"identification_type_code"`
	DepartmentCode              string             `json:"department_code"`
	MunicipalityCode            string             `json:"municipality_code"`
	AddressLine                 string             `json:"address_line"`
	Email                       string             `json:"email"`
	Phone                       string             `json:"phone"`
	Environment                 domain.Environment `json:"environment"`
	EntityTypeCode              string             `json:"entity_type_code"`
	TaxSchemeCode               string             `json:"tax_scheme_code"`
	TaxSchemeName               string             `json:"tax_scheme_name"`
	LiabilityCodes              []string           `json:"liability_codes"`
	TaxRegimeCode               *string            `json:"tax_regime_code"`
	IndustryClassificationCodes []string           `json:"industry_classification_codes"`
	MerchantRegistrationNumber  *string            `json:"merchant_registration_number"`
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*domain.Company, error) {
	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.BusinessName = req.BusinessName
	existing.TradeName = req.TradeName
	existing.IdentificationTypeCode = req.IdentificationTypeCode
	existing.DepartmentCode = req.DepartmentCode
	existing.MunicipalityCode = req.MunicipalityCode
	existing.AddressLine = req.AddressLine
	existing.Email = req.Email
	existing.Phone = req.Phone
	existing.Environment = req.Environment
	existing.EntityTypeCode = req.EntityTypeCode
	existing.TaxSchemeCode = req.TaxSchemeCode
	existing.TaxSchemeName = req.TaxSchemeName
	existing.LiabilityCodes = req.LiabilityCodes
	existing.TaxRegimeCode = req.TaxRegimeCode
	existing.IndustryClassificationCodes = req.IndustryClassificationCodes
	existing.MerchantRegistrationNumber = req.MerchantRegistrationNumber

	return uc.repo.UpdateProfile(ctx, *existing)
}
