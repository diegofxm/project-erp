package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	accountingdomain "github.com/diegofxm/erp/internal/accounting/domain"
	"github.com/diegofxm/erp/internal/purchase/domain"
)

// AddWithholdingUseCase aplica una retención a una orden de compra confirmada (antes de
// recibirla). Lee el concepto directamente del catálogo de accounting (misma empresa no
// aplica — el PUC/conceptos son globales) para tomar tarifa y cuenta contable, y guarda una
// foto de esos datos en la orden — mismo patrón de import directo entre módulos que usa
// electronic/from_purchase.go con supplierdomain.Repository.
type AddWithholdingUseCase struct {
	purchases    domain.Repository
	withholdings domain.WithholdingRepository
	concepts     accountingdomain.WithholdingConceptRepository
}

func NewAddWithholdingUseCase(
	purchases domain.Repository,
	withholdings domain.WithholdingRepository,
	concepts accountingdomain.WithholdingConceptRepository,
) *AddWithholdingUseCase {
	return &AddWithholdingUseCase{purchases: purchases, withholdings: withholdings, concepts: concepts}
}

type AddWithholdingRequest struct {
	PurchaseID uuid.UUID
	ConceptID  uuid.UUID
	Base       float64
}

func (uc *AddWithholdingUseCase) Execute(ctx context.Context, companyID uuid.UUID, req AddWithholdingRequest) (*domain.PurchaseWithholding, error) {
	if req.Base <= 0 {
		return nil, fmt.Errorf("la base debe ser mayor a cero")
	}

	order, err := uc.purchases.GetByID(ctx, companyID, req.PurchaseID)
	if err != nil {
		return nil, err
	}
	if order.Status != domain.StatusConfirmed {
		return nil, domain.ErrPurchaseNotConfirmed
	}

	concept, err := uc.concepts.GetByID(ctx, req.ConceptID)
	if err != nil {
		return nil, err
	}

	amount := req.Base * float64(concept.RateBP) / 10000

	w := domain.PurchaseWithholding{
		PurchaseOrderID: order.ID,
		ConceptCode:     concept.Code,
		ConceptName:     concept.Name,
		Base:            req.Base,
		RateBP:          concept.RateBP,
		Amount:          amount,
		AccountPayable:  concept.AccountPayable,
	}
	return uc.withholdings.Add(ctx, w)
}

func (uc *AddWithholdingUseCase) List(ctx context.Context, companyID, purchaseID uuid.UUID) ([]domain.PurchaseWithholding, error) {
	if _, err := uc.purchases.GetByID(ctx, companyID, purchaseID); err != nil {
		return nil, err
	}
	return uc.withholdings.ListByPurchase(ctx, purchaseID)
}
