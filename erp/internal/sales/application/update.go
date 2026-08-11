package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
)

// UpdateUseCase corrige cliente/fecha/notas/líneas de una venta EN BORRADOR -- el repositorio
// valida el estado (Repository.Update). Mismo request que crear (CreateRequest/LineRequest,
// definidos en create.go), porque el formulario de captura es el mismo antes y después de
// guardar mientras siga en borrador.
type UpdateUseCase struct {
	repo domain.Repository
}

func NewUpdateUseCase(repo domain.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

func (uc *UpdateUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, req CreateRequest) (*domain.Sale, error) {
	lines := make([]domain.SaleLine, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = domain.SaleLine{
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

	s := domain.Sale{
		CustomerID: req.CustomerID,
		IssueDate:  issueDate,
		DueDate:    req.DueDate,
		Notes:      req.Notes,
		Lines:      lines,
	}
	s.CalculateTotals()

	return uc.repo.Update(ctx, companyID, id, s)
}
