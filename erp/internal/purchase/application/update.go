package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

// UpdateUseCase corrige proveedor/fecha/notas/líneas de una orden EN BORRADOR -- el repositorio
// valida el estado (Repository.Update). Mismo request que crear (CreateRequest/LineRequest,
// definidos en create.go).
type UpdateUseCase struct {
	repo domain.Repository
}

func NewUpdateUseCase(repo domain.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

func (uc *UpdateUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, req CreateRequest) (*domain.PurchaseOrder, error) {
	lines := make([]domain.PurchaseLine, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = domain.PurchaseLine{
			ID:          uuid.New(),
			ProductID:   l.ProductID,
			Description: l.Description,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			Discount:    l.Discount,
			TaxRate:     l.TaxRate,
		}
	}

	issueDate := req.IssueDate
	if issueDate.IsZero() {
		issueDate = time.Now()
	}

	o := domain.PurchaseOrder{
		SupplierID:   req.SupplierID,
		IssueDate:    issueDate,
		DueDate:      req.DueDate,
		Notes:        req.Notes,
		Lines:        lines,
		PaymentMeans: paymentMeansToCofdom(req.PaymentMeans),
	}
	o.CalculateTotals()

	return uc.repo.Update(ctx, companyID, id, o)
}
