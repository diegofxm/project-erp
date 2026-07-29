package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/product/domain"
)

type UpdateUseCase struct {
	repo domain.Repository
}

func NewUpdateUseCase(repo domain.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

func (uc *UpdateUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, req SaveRequest) (*domain.Product, error) {
	existing, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}

	existing.Code = req.Code
	existing.Name = req.Name
	existing.Description = req.Description
	existing.UnitMeasureCode = req.UnitMeasureCode
	existing.StandardCode = req.StandardCode
	existing.StandardCodeType = req.StandardCodeType
	existing.StandardCodeID = req.StandardCodeID
	existing.StandardCodeAgencyID = req.StandardCodeAgencyID
	existing.IsService = req.IsService
	existing.TaxSchemeCode = req.TaxSchemeCode
	existing.TaxSchemeName = req.TaxSchemeName
	existing.TaxRate = req.TaxRate
	existing.BasePrice = req.BasePrice

	return uc.repo.Update(ctx, *existing)
}
