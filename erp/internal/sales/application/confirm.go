package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type ConfirmUseCase struct {
	repo domain.Repository
	bus  *events.Bus
}

func NewConfirmUseCase(repo domain.Repository, bus *events.Bus) *ConfirmUseCase {
	return &ConfirmUseCase{repo: repo, bus: bus}
}

func (uc *ConfirmUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) (*domain.Sale, error) {
	s, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if s.Status != domain.StatusDraft {
		return nil, domain.ErrSaleNotDraft
	}
	if err := uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusConfirmed); err != nil {
		return nil, err
	}
	s.Status = domain.StatusConfirmed

	_, tax, total := s.GrandTotal()
	uc.bus.Publish(domain.SaleConfirmed{
		SaleID:     s.ID,
		CompanyID:  s.CompanyID,
		CustomerID: s.CustomerID,
		Total:      total,
		TaxAmount:  tax,
		IssueDate:  s.IssueDate,
	})

	return s, nil
}
