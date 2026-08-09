package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type ReceiveUseCase struct {
	repo  domain.Repository
	bus   *events.Bus
	audit events.AuditLogger
}

func NewReceiveUseCase(repo domain.Repository, bus *events.Bus, audit events.AuditLogger) *ReceiveUseCase {
	return &ReceiveUseCase{repo: repo, bus: bus, audit: audit}
}

func (uc *ReceiveUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) (*domain.PurchaseOrder, error) {
	o, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if o.Status != domain.StatusConfirmed {
		return nil, domain.ErrPurchaseNotConfirmed
	}
	if err := uc.repo.UpdateStatus(ctx, companyID, id, domain.StatusReceived); err != nil {
		return nil, err
	}
	o.Status = domain.StatusReceived

	var taxAmount float64
	for _, l := range o.Lines {
		taxAmount += l.TaxAmount
	}
	events.PublishAndAudit(ctx, uc.bus, domain.PurchaseReceived{
		PurchaseID:   o.ID,
		CompanyID:    o.CompanyID,
		SupplierID:   o.SupplierID,
		Number:       o.Number,
		Total:        o.GrandTotal(),
		TaxAmount:    taxAmount,
		IssueDate:    o.IssueDate,
		Lines:        o.Lines,
		Withholdings: o.Withholdings,
	}, uc.audit, o.CompanyID, "purchase.received_side_effect_failed", "purchase", o.ID)

	return o, nil
}
