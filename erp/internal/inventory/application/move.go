package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/inventory/domain"
)

type MoveUseCase struct {
	repo domain.Repository
}

func NewMoveUseCase(repo domain.Repository) *MoveUseCase {
	return &MoveUseCase{repo: repo}
}

type MoveRequest struct {
	ProductID   uuid.UUID           `json:"product_id"`
	Warehouse   string              `json:"warehouse"`
	Type        domain.MovementType `json:"type"`
	Quantity    float64             `json:"quantity"`
	Reference   string              `json:"reference"`
	Description string              `json:"description"`
}

func (uc *MoveUseCase) Execute(ctx context.Context, companyID uuid.UUID, req MoveRequest) (*domain.Movement, error) {
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("la cantidad debe ser mayor a cero")
	}

	// Para salidas: verificar stock suficiente
	if req.Type == domain.MovementExit {
		stock, err := uc.repo.GetStock(ctx, companyID, req.ProductID, req.Warehouse)
		if err != nil {
			return nil, domain.ErrInsufficientStock
		}
		if stock.Quantity < req.Quantity {
			return nil, domain.ErrInsufficientStock
		}
	}

	m := domain.Movement{
		ID:          uuid.New(),
		CompanyID:   companyID,
		ProductID:   req.ProductID,
		Warehouse:   req.Warehouse,
		Type:        req.Type,
		Quantity:    req.Quantity,
		Reference:   req.Reference,
		Description: req.Description,
	}

	saved, err := uc.repo.SaveMovement(ctx, m)
	if err != nil {
		return nil, err
	}

	// Actualizar stock
	current, err := uc.repo.GetStock(ctx, companyID, req.ProductID, req.Warehouse)
	var currentQty float64
	if err == nil {
		currentQty = current.Quantity
	}

	var newQty float64
	switch req.Type {
	case domain.MovementEntry, domain.MovementAdjust:
		newQty = currentQty + req.Quantity
	case domain.MovementExit:
		newQty = currentQty - req.Quantity
	case domain.MovementTransfer:
		// El traslado genera dos movimientos; aquí solo se procesa uno a la vez
		newQty = currentQty - req.Quantity
	}

	entry := domain.StockEntry{
		ID:        uuid.New(),
		CompanyID: companyID,
		ProductID: req.ProductID,
		Warehouse: req.Warehouse,
		Quantity:  newQty,
	}
	if current != nil {
		entry.ID = current.ID
	}
	if err := uc.repo.UpsertStock(ctx, entry); err != nil {
		return nil, err
	}

	return saved, nil
}
