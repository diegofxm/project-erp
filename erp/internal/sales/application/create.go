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
	IssueDate    string               `json:"issue_date"` // YYYY-MM-DD -- ver parseDate más abajo
	DueDate      string               `json:"due_date"`   // YYYY-MM-DD, opcional
	Notes        string               `json:"notes"`
	Lines        []LineRequest        `json:"lines"`
	PaymentMeans []PaymentMeanRequest `json:"payment_means"`
}

// parseDate interpreta una fecha YYYY-MM-DD tal como la manda <input type="date"> del frontend
// -- NUNCA time.Time directo en el DTO: el JSON de Go exige RFC3339 completo para time.Time
// ("2026-08-11T00:00:00Z"), y decodificar "2026-08-11" a secas contra ese tipo falla con 400
// "cuerpo inválido" antes de que el handler llegue a validar nada (bug real encontrado
// 2026-08-11 -- crear una venta/orden directamente, sin pasar por cotización, siempre fallaba).
// Cadena vacía o no parseable → zero time.Time, el llamador decide el default.
func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
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

	issueDate := parseDate(req.IssueDate)
	if issueDate.IsZero() {
		issueDate = time.Now()
	}
	seq, err := uc.repo.NextSaleNumber(ctx, companyID, issueDate.Year())
	if err != nil {
		return nil, fmt.Errorf("asignar consecutivo: %w", err)
	}

	var dueDate *time.Time
	if d := parseDate(req.DueDate); !d.IsZero() {
		dueDate = &d
	}

	s := domain.Sale{
		ID:           uuid.New(),
		CompanyID:    companyID,
		CustomerID:   req.CustomerID,
		Number:       fmt.Sprintf("VTA-%d-%05d", issueDate.Year(), seq),
		Status:       domain.StatusDraft,
		IssueDate:    issueDate,
		DueDate:      dueDate,
		Notes:        req.Notes,
		Lines:        lines,
		PaymentMeans: paymentMeansToCofdom(req.PaymentMeans),
	}
	s.CalculateTotals()

	return uc.repo.Save(ctx, s)
}
