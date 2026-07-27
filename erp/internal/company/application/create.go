package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

// MembershipLinker vincula el usuario creador a la empresa recién creada.
// Implementado por security/infrastructure/persistence/postgres.Repository.
type MembershipLinker interface {
	AddCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error
}

type CreateUseCase struct {
	repo    domain.Repository
	members MembershipLinker
}

func NewCreateUseCase(repo domain.Repository, members MembershipLinker) *CreateUseCase {
	return &CreateUseCase{repo: repo, members: members}
}

type CreateRequest struct {
	NIT                         string
	CheckDigit                  string
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

func (uc *CreateUseCase) Execute(ctx context.Context, creatorID uuid.UUID, req CreateRequest) (*domain.Company, error) {
	if _, err := uc.repo.GetByNIT(ctx, req.NIT); err == nil {
		return nil, domain.ErrNITTaken
	}

	if req.Environment == "" {
		req.Environment = domain.Habilitacion
	}
	if req.EntityTypeCode == "" {
		req.EntityTypeCode = "1"
	}
	if req.TaxSchemeCode == "" {
		req.TaxSchemeCode = "ZZ"
		req.TaxSchemeName = "No aplica"
	}

	c := domain.Company{
		ID:                          uuid.New(),
		NIT:                         req.NIT,
		CheckDigit:                  req.CheckDigit,
		BusinessName:                req.BusinessName,
		TradeName:                   req.TradeName,
		IdentificationTypeCode:      req.IdentificationTypeCode,
		DepartmentCode:              req.DepartmentCode,
		MunicipalityCode:            req.MunicipalityCode,
		AddressLine:                 req.AddressLine,
		Email:                       req.Email,
		Phone:                       req.Phone,
		Environment:                 req.Environment,
		EntityTypeCode:              req.EntityTypeCode,
		TaxSchemeCode:               req.TaxSchemeCode,
		TaxSchemeName:               req.TaxSchemeName,
		LiabilityCodes:              req.LiabilityCodes,
		TaxRegimeCode:               req.TaxRegimeCode,
		IndustryClassificationCodes: req.IndustryClassificationCodes,
		MerchantRegistrationNumber:  req.MerchantRegistrationNumber,
		IsActive:                    true,
	}

	saved, err := uc.repo.Save(ctx, c)
	if err != nil {
		return nil, err
	}

	if err := uc.members.AddCompany(ctx, creatorID, saved.ID, "owner"); err != nil {
		return nil, err
	}

	return saved, nil
}
