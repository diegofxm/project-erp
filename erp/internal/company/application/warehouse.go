package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

type WarehouseUseCase struct {
	repo domain.WarehouseRepository
}

func NewWarehouseUseCase(repo domain.WarehouseRepository) *WarehouseUseCase {
	return &WarehouseUseCase{repo: repo}
}

type CreateWarehouseRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (uc *WarehouseUseCase) Create(ctx context.Context, companyID uuid.UUID, req CreateWarehouseRequest) (*domain.Warehouse, error) {
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("código y nombre son requeridos")
	}
	w := domain.Warehouse{
		ID:        uuid.New(),
		CompanyID: companyID,
		Code:      req.Code,
		Name:      req.Name,
		Address:   req.Address,
		IsActive:  true,
	}
	return uc.repo.Save(ctx, w)
}

func (uc *WarehouseUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.Warehouse, error) {
	return uc.repo.List(ctx, companyID)
}

func (uc *WarehouseUseCase) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Warehouse, error) {
	return uc.repo.GetByID(ctx, companyID, id)
}

type UpdateWarehouseRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (uc *WarehouseUseCase) Update(ctx context.Context, companyID, id uuid.UUID, req UpdateWarehouseRequest) (*domain.Warehouse, error) {
	w, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if req.Code != "" {
		w.Code = req.Code
	}
	if req.Name != "" {
		w.Name = req.Name
	}
	w.Address = req.Address
	return uc.repo.Update(ctx, *w)
}

func (uc *WarehouseUseCase) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.repo.Deactivate(ctx, companyID, id)
}
