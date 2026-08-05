package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type PaymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

type RecordPaymentRequest struct {
	CompanyID      uuid.UUID
	SubscriptionID *uuid.UUID
	Type           domain.PaymentType
	AmountCents    int64
	Note           string
	PaidAt         time.Time
}

func (uc *PaymentUseCase) Record(ctx context.Context, req RecordPaymentRequest) (*domain.Payment, error) {
	paidAt := req.PaidAt
	if paidAt.IsZero() {
		paidAt = time.Now()
	}
	return uc.repo.Create(ctx, domain.Payment{
		CompanyID: req.CompanyID, SubscriptionID: req.SubscriptionID, Type: req.Type,
		AmountCents: req.AmountCents, Note: req.Note, PaidAt: paidAt,
	})
}

func (uc *PaymentUseCase) ListByCompany(ctx context.Context, companyID uuid.UUID) ([]domain.Payment, error) {
	return uc.repo.ListByCompany(ctx, companyID)
}
