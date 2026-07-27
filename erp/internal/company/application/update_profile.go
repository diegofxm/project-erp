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
	BusinessName                string
	TradeName                   string
	IdentificationTypeCode      string
	DepartmentCode              string
	MunicipalityCode            string
	AddressLine                 string
	Email                       string
	Phone                       string
	Environment                 domain.Environment
	EntityTypeCode              string
	TaxSchemeCode               string
	TaxSchemeName               string
	LiabilityCodes              []string
	TaxRegimeCode               *string
	IndustryClassificationCodes []string
	MerchantRegistrationNumber  *string
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
