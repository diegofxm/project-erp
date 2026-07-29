package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/nit"
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
	NIT                         string             `json:"nit"`
	CheckDigit                  string             `json:"check_digit"`
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

	// Para NITs (tipo 31) el DV se calcula siempre — aunque el usuario mande uno incorrecto,
	// la DIAN requiere el dígito exacto o rechaza el documento.
	checkDigit := req.CheckDigit
	if req.IdentificationTypeCode == "31" {
		if dv, err := nit.ComputeCheckDigit(req.NIT); err == nil {
			checkDigit = dv
		}
	}

	c := domain.Company{
		ID:                          uuid.New(),
		NIT:                         req.NIT,
		CheckDigit:                  checkDigit,
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
