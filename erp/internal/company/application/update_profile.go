package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/nit"
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

	// liability_codes e industry_classification_codes son TEXT[] NOT NULL en el esquema — un
	// payload que omita esas claves decodifica a nil (no a []string{}), y guardar nil ahí
	// revienta la restricción NOT NULL en vez de dar un error legible. PUT reemplaza el recurso
	// completo (igual que el resto del proyecto), así que "no mandaste el campo" se trata como
	// "sin códigos", no como "deja lo que había".
	if req.LiabilityCodes == nil {
		req.LiabilityCodes = []string{}
	}
	if req.IndustryClassificationCodes == nil {
		req.IndustryClassificationCodes = []string{}
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

	// Recalcula el DV siempre que sea NIT (tipo 31) — garantiza que el dígito almacenado
	// sea el correcto aunque el NIT original se registró con un DV errado.
	if existing.IdentificationTypeCode == "31" {
		if dv, err := nit.ComputeCheckDigit(existing.NIT); err == nil {
			existing.CheckDigit = dv
		}
	}

	return uc.repo.UpdateProfile(ctx, *existing)
}
