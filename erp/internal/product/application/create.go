package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/product/domain"
)

type CreateUseCase struct {
	repo domain.Repository
}

func NewCreateUseCase(repo domain.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

type SaveRequest struct {
	Code                  string  `json:"code"`
	Name                  string  `json:"name"`
	Description           string  `json:"description"`
	UnitMeasureCode       string  `json:"unit_measure_code"`
	StandardCode          string  `json:"standard_code"`
	StandardCodeType      string  `json:"standard_code_type"`
	StandardCodeID        string  `json:"standard_code_id"`
	StandardCodeAgencyID  string  `json:"standard_code_agency_id"`
	IsService             bool    `json:"is_service"`
	TaxSchemeCode         string  `json:"tax_scheme_code"`
	TaxSchemeName         string  `json:"tax_scheme_name"`
	TaxRate               float64 `json:"tax_rate"`
	BasePrice             float64 `json:"base_price"`
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req SaveRequest) (*domain.Product, error) {
	_, err := uc.repo.GetByCode(ctx, companyID, req.Code)
	if err == nil {
		return nil, domain.ErrDuplicateProduct
	}

	p := domain.Product{
		ID:                   uuid.New(),
		CompanyID:            companyID,
		Code:                 req.Code,
		Name:                 req.Name,
		Description:          req.Description,
		UnitMeasureCode:      req.UnitMeasureCode,
		StandardCode:         req.StandardCode,
		StandardCodeType:     req.StandardCodeType,
		StandardCodeID:       req.StandardCodeID,
		StandardCodeAgencyID: req.StandardCodeAgencyID,
		IsService:            req.IsService,
		TaxSchemeCode:        req.TaxSchemeCode,
		TaxSchemeName:        req.TaxSchemeName,
		TaxRate:              req.TaxRate,
		BasePrice:            req.BasePrice,
		IsActive:             true,
	}

	return uc.repo.Save(ctx, p)
}
