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
	Code             string
	Name             string
	Description      string
	UnitMeasureCode  string
	StandardCode     string
	StandardCodeType string
	IsService        bool
	TaxSchemeCode    string
	TaxSchemeName    string
	TaxRate          float64
	BasePrice        float64
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req SaveRequest) (*domain.Product, error) {
	_, err := uc.repo.GetByCode(ctx, companyID, req.Code)
	if err == nil {
		return nil, domain.ErrDuplicateProduct
	}

	p := domain.Product{
		ID:               uuid.New(),
		CompanyID:        companyID,
		Code:             req.Code,
		Name:             req.Name,
		Description:      req.Description,
		UnitMeasureCode:  req.UnitMeasureCode,
		StandardCode:     req.StandardCode,
		StandardCodeType: req.StandardCodeType,
		IsService:        req.IsService,
		TaxSchemeCode:    req.TaxSchemeCode,
		TaxSchemeName:    req.TaxSchemeName,
		TaxRate:          req.TaxRate,
		BasePrice:        req.BasePrice,
		IsActive:         true,
	}

	return uc.repo.Save(ctx, p)
}
