package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
)

type CreateUseCase struct {
	repo domain.Repository
}

func NewCreateUseCase(repo domain.Repository) *CreateUseCase {
	return &CreateUseCase{repo: repo}
}

type LineRequest struct {
	ProductID   uuid.UUID `json:"product_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	TaxRate     float64   `json:"tax_rate"`
}

type CreateRequest struct {
	SupplierID uuid.UUID    `json:"supplier_id"`
	Number     string       `json:"number"`
	IssueDate  time.Time    `json:"issue_date"`
	DueDate    *time.Time   `json:"due_date"`
	Notes      string       `json:"notes"`
	Lines      []LineRequest `json:"lines"`
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req CreateRequest) (*domain.PurchaseOrder, error) {
	lines := make([]domain.PurchaseLine, len(req.Lines))
	for i, l := range req.Lines {
		lines[i] = domain.PurchaseLine{
			ID:          uuid.New(),
			ProductID:   l.ProductID,
			Description: l.Description,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			TaxRate:     l.TaxRate,
		}
	}

	o := domain.PurchaseOrder{
		ID:         uuid.New(),
		CompanyID:  companyID,
		SupplierID: req.SupplierID,
		Number:     req.Number,
		Status:     domain.StatusDraft,
		IssueDate:  req.IssueDate,
		DueDate:    req.DueDate,
		Notes:      req.Notes,
		Lines:      lines,
	}
	o.CalculateTotals()

	return uc.repo.Save(ctx, o)
}
