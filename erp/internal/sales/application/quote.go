package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
)

type QuoteUseCase struct {
	quoteRepo domain.QuoteRepository
	saleRepo  domain.Repository
}

func NewQuoteUseCase(quoteRepo domain.QuoteRepository, saleRepo domain.Repository) *QuoteUseCase {
	return &QuoteUseCase{quoteRepo: quoteRepo, saleRepo: saleRepo}
}

type CreateQuoteRequest struct {
	CustomerID uuid.UUID         `json:"customer_id"`
	Lines      []QuoteLine       `json:"lines"`
	ValidUntil string            `json:"valid_until"` // YYYY-MM-DD, opcional
	Notes      string            `json:"notes"`
}

type QuoteLine struct {
	ProductID   uuid.UUID `json:"product_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	TaxRate     float64   `json:"tax_rate"`
}

func (uc *QuoteUseCase) Create(ctx context.Context, companyID uuid.UUID, req CreateQuoteRequest) (*domain.Quote, error) {
	if len(req.Lines) == 0 {
		return nil, fmt.Errorf("la cotización debe tener al menos una línea")
	}
	q := domain.Quote{
		ID:         uuid.New(),
		CompanyID:  companyID,
		CustomerID: req.CustomerID,
		Status:     domain.QuoteStatusDraft,
		IssueDate:  time.Now(),
		Notes:      req.Notes,
	}
	if req.ValidUntil != "" {
		t, err := time.Parse("2006-01-02", req.ValidUntil)
		if err == nil {
			q.ValidUntil = &t
		}
	}
	for _, l := range req.Lines {
		q.Lines = append(q.Lines, domain.QuoteLine{
			ProductID:   l.ProductID,
			Description: l.Description,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			Discount:    l.Discount,
			TaxRate:     l.TaxRate,
		})
	}
	q.CalculateTotals()
	return uc.quoteRepo.Save(ctx, q)
}

func (uc *QuoteUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.Quote, error) {
	return uc.quoteRepo.List(ctx, companyID)
}

func (uc *QuoteUseCase) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Quote, error) {
	return uc.quoteRepo.GetByID(ctx, companyID, id)
}

func (uc *QuoteUseCase) Send(ctx context.Context, companyID, id uuid.UUID) (*domain.Quote, error) {
	q, err := uc.quoteRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.QuoteStatusDraft {
		return nil, domain.ErrQuoteNotDraft
	}
	if err := uc.quoteRepo.UpdateStatus(ctx, companyID, id, domain.QuoteStatusSent); err != nil {
		return nil, err
	}
	q.Status = domain.QuoteStatusSent
	return q, nil
}

func (uc *QuoteUseCase) Accept(ctx context.Context, companyID, id uuid.UUID) (*domain.Quote, error) {
	q, err := uc.quoteRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.QuoteStatusSent {
		return nil, domain.ErrQuoteNotSent
	}
	if err := uc.quoteRepo.UpdateStatus(ctx, companyID, id, domain.QuoteStatusAccepted); err != nil {
		return nil, err
	}
	q.Status = domain.QuoteStatusAccepted
	return q, nil
}

func (uc *QuoteUseCase) Reject(ctx context.Context, companyID, id uuid.UUID) error {
	q, err := uc.quoteRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if q.Status != domain.QuoteStatusSent {
		return domain.ErrQuoteNotSent
	}
	return uc.quoteRepo.UpdateStatus(ctx, companyID, id, domain.QuoteStatusRejected)
}

// ConvertToSale crea una venta a partir de una cotización aceptada.
func (uc *QuoteUseCase) ConvertToSale(ctx context.Context, companyID, id uuid.UUID) (*domain.Sale, error) {
	q, err := uc.quoteRepo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.QuoteStatusAccepted {
		return nil, domain.ErrQuoteNotAccepted
	}

	sale := domain.Sale{
		ID:         uuid.New(),
		CompanyID:  companyID,
		CustomerID: q.CustomerID,
		Status:     domain.StatusDraft,
		IssueDate:  time.Now(),
		Notes:      q.Notes,
	}
	for _, ql := range q.Lines {
		sale.Lines = append(sale.Lines, domain.SaleLine{
			ProductID:   ql.ProductID,
			Description: ql.Description,
			Quantity:    ql.Quantity,
			UnitPrice:   ql.UnitPrice,
			Discount:    ql.Discount,
			TaxRate:     ql.TaxRate,
		})
	}
	sale.CalculateTotals()
	return uc.saleRepo.Save(ctx, sale)
}

func (uc *QuoteUseCase) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.quoteRepo.Delete(ctx, companyID, id)
}
