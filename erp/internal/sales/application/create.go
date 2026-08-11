package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/erp/internal/sales/domain"
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
	Discount    float64   `json:"discount"`
	TaxRate     float64   `json:"tax_rate"`
}

// PaymentMeanRequest -- mismos campos/catálogos DIAN (payment_terms/payment_methods) que ya usa
// electronic para documentos electrónicos; se convierte 1:1 a cofdom.PaymentMean.
type PaymentMeanRequest struct {
	Code              string `json:"code"`
	PaymentMethodCode string `json:"payment_method_code"`
	DueDate           string `json:"due_date,omitempty"`
	PaymentReference  string `json:"payment_reference,omitempty"`
}

func paymentMeansToCofdom(pms []PaymentMeanRequest) []cofdom.PaymentMean {
	out := make([]cofdom.PaymentMean, len(pms))
	for i, pm := range pms {
		out[i] = cofdom.PaymentMean{
			Code: pm.Code, PaymentMethodCode: pm.PaymentMethodCode,
			DueDate: pm.DueDate, PaymentReference: pm.PaymentReference,
		}
	}
	return out
}

type CreateRequest struct {
	CustomerID   uuid.UUID            `json:"customer_id"`
	IssueDate    time.Time            `json:"issue_date"`
	DueDate      *time.Time           `json:"due_date"`
	Notes        string               `json:"notes"`
	Lines        []LineRequest        `json:"lines"`
	PaymentMeans []PaymentMeanRequest `json:"payment_means"`
}

func (uc *CreateUseCase) Execute(ctx context.Context, companyID uuid.UUID, req CreateRequest) (*domain.Sale, error) {
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

	year := req.IssueDate.Year()
	if year == 0 {
		year = time.Now().Year()
	}
	seq, err := uc.repo.NextSaleNumber(ctx, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("asignar consecutivo: %w", err)
	}

	s := domain.Sale{
		ID:           uuid.New(),
		CompanyID:    companyID,
		CustomerID:   req.CustomerID,
		Number:       fmt.Sprintf("VTA-%d-%05d", year, seq),
		Status:       domain.StatusDraft,
		IssueDate:    req.IssueDate,
		DueDate:      req.DueDate,
		Notes:        req.Notes,
		Lines:        lines,
		PaymentMeans: paymentMeansToCofdom(req.PaymentMeans),
	}
	s.CalculateTotals()

	return uc.repo.Save(ctx, s)
}
