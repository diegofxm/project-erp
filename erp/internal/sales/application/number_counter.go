package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
)

// SetNumberCounterUseCase permite fijar el próximo consecutivo de venta o cotización — pensado
// para migrar una empresa que ya traía su propia numeración de otro sistema (ver
// electronic.NumberingRange, que resuelve el mismo problema para los rangos de resolución DIAN).
type SetNumberCounterUseCase struct {
	saleRepo  domain.Repository
	quoteRepo domain.QuoteRepository
}

func NewSetNumberCounterUseCase(saleRepo domain.Repository, quoteRepo domain.QuoteRepository) *SetNumberCounterUseCase {
	return &SetNumberCounterUseCase{saleRepo: saleRepo, quoteRepo: quoteRepo}
}

func (uc *SetNumberCounterUseCase) SetSale(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error) {
	if nextNumber < 1 {
		return 0, domain.ErrNumberCounterInvalid
	}
	return uc.saleRepo.SetSaleNumberCounter(ctx, companyID, year, nextNumber)
}

func (uc *SetNumberCounterUseCase) SetQuote(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error) {
	if nextNumber < 1 {
		return 0, domain.ErrNumberCounterInvalid
	}
	return uc.quoteRepo.SetQuoteNumberCounter(ctx, companyID, year, nextNumber)
}
