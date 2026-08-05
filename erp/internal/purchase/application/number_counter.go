package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

// SetNumberCounterUseCase permite fijar el próximo consecutivo de orden de compra — pensado para
// migrar una empresa que ya traía su propia numeración de otro sistema (ver
// sales.application.SetNumberCounterUseCase).
type SetNumberCounterUseCase struct {
	repo domain.Repository
}

func NewSetNumberCounterUseCase(repo domain.Repository) *SetNumberCounterUseCase {
	return &SetNumberCounterUseCase{repo: repo}
}

func (uc *SetNumberCounterUseCase) Execute(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error) {
	if nextNumber < 1 {
		return 0, domain.ErrNumberCounterInvalid
	}
	return uc.repo.SetPurchaseNumberCounter(ctx, companyID, year, nextNumber)
}
