package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/customer/domain"
	"github.com/diegofxm/erp/internal/shared/nit"
)

type UpdateUseCase struct {
	repo domain.Repository
}

func NewUpdateUseCase(repo domain.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

func (uc *UpdateUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, req SaveRequest) (*domain.Customer, error) {
	existing, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}

	checkDigit := req.CheckDigit
	if req.IdentificationTypeCode == "31" {
		if dv, err := nit.ComputeCheckDigit(req.IdentificationNumber); err == nil {
			checkDigit = dv
		}
	}

	existing.IdentificationTypeCode = req.IdentificationTypeCode
	existing.IdentificationNumber = req.IdentificationNumber
	existing.CheckDigit = checkDigit
	existing.EntityTypeCode = req.EntityTypeCode
	existing.MerchantRegistrationNumber = req.MerchantRegistrationNumber
	existing.Name = req.Name
	existing.TaxSchemeCode = req.TaxSchemeCode
	existing.TaxSchemeName = req.TaxSchemeName
	existing.TaxRegimeCode = req.TaxRegimeCode
	existing.LiabilityCodes = req.LiabilityCodes
	existing.DepartmentCode = req.DepartmentCode
	existing.MunicipalityCode = req.MunicipalityCode
	existing.AddressLine = req.AddressLine
	existing.AddressCityName = req.AddressCityName
	existing.AddressStateName = req.AddressStateName
	existing.AddressCountryCode = req.AddressCountryCode
	existing.AddressCountryName = req.AddressCountryName
	existing.Email = req.Email
	existing.Phone = req.Phone
	existing.CreditLimit = req.CreditLimit

	return uc.repo.Update(ctx, *existing)
}
