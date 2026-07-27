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
	IdentificationTypeCode string
	IdentificationNumber   string
	CheckDigit             string
	Name                   string
	TaxSchemeCode          string
	TaxSchemeName          string
	TaxRegimeCode          *string
	LiabilityCodes         []string
	DepartmentCode         string
	MunicipalityCode       string
	AddressLine            string
	Email                  string
	Phone                  string
	PaymentTermsDays       int
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req SaveRequest) (*domain.Supplier, error) {
	_, err := uc.repo.GetByIdentification(ctx, companyID, req.IdentificationTypeCode, req.IdentificationNumber)
	if err == nil {
		return nil, domain.ErrDuplicateSupplier
	}

	s := domain.Supplier{
		ID:                     uuid.New(),
		CompanyID:              companyID,
		IdentificationTypeCode: req.IdentificationTypeCode,
		IdentificationNumber:   req.IdentificationNumber,
		CheckDigit:             req.CheckDigit,
		Name:                   req.Name,
		TaxSchemeCode:          req.TaxSchemeCode,
		TaxSchemeName:          req.TaxSchemeName,
		TaxRegimeCode:          req.TaxRegimeCode,
		LiabilityCodes:         req.LiabilityCodes,
		DepartmentCode:         req.DepartmentCode,
		MunicipalityCode:       req.MunicipalityCode,
		AddressLine:            req.AddressLine,
		Email:                  req.Email,
		Phone:                  req.Phone,
		PaymentTermsDays:       req.PaymentTermsDays,
		IsActive:               true,
	}

	return uc.repo.Save(ctx, s)
}
