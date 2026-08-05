package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type PaymentUseCase struct {
	paymentRepo  domain.PaymentRepository
	purchaseRepo domain.Repository
	bus          *events.Bus
}

func NewPaymentUseCase(paymentRepo domain.PaymentRepository, purchaseRepo domain.Repository, bus *events.Bus) *PaymentUseCase {
	return &PaymentUseCase{paymentRepo: paymentRepo, purchaseRepo: purchaseRepo, bus: bus}
}

type RecordPaymentRequest struct {
	PurchaseID    uuid.UUID `json:"purchase_id"`
	PaymentDate   string    `json:"payment_date"` // YYYY-MM-DD
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Reference     string    `json:"reference"`
	Notes         string    `json:"notes"`
}

func (uc *PaymentUseCase) Record(ctx context.Context, companyID uuid.UUID, req RecordPaymentRequest) (*domain.PurchasePayment, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	purchase, err := uc.purchaseRepo.GetByID(ctx, companyID, req.PurchaseID)
	if err != nil {
		return nil, err
	}
	if purchase.Status != domain.StatusReceived {
		return nil, fmt.Errorf("solo se pueden registrar pagos sobre órdenes recibidas")
	}

	date := time.Now()
	if req.PaymentDate != "" {
		if t, err := time.Parse("2006-01-02", req.PaymentDate); err == nil {
			date = t
		}
	}

	method := domain.PaymentMethod(req.PaymentMethod)
	if method == "" {
		method = domain.PaymentTransfer
	}

	p := domain.PurchasePayment{
		ID:            uuid.New(),
		CompanyID:     companyID,
		PurchaseID:    req.PurchaseID,
		PaymentDate:   date,
		Amount:        req.Amount,
		PaymentMethod: method,
		Reference:     req.Reference,
		Notes:         req.Notes,
	}
	saved, err := uc.paymentRepo.Save(ctx, p)
	if err != nil {
		return nil, err
	}

	uc.bus.Publish(domain.PurchasePaymentRecorded{
		PaymentID:      saved.ID,
		CompanyID:      companyID,
		PurchaseID:     saved.PurchaseID,
		PurchaseNumber: purchase.Number,
		Amount:         saved.Amount,
		PaymentMethod: saved.PaymentMethod,
		PaymentDate:   saved.PaymentDate,
	})

	return saved, nil
}

func (uc *PaymentUseCase) ListByPurchase(ctx context.Context, companyID, purchaseID uuid.UUID) ([]domain.PurchasePayment, error) {
	return uc.paymentRepo.ListByPurchase(ctx, companyID, purchaseID)
}

// GetPayables devuelve las órdenes de compra recibidas con saldo pendiente (cuentas por pagar).
func (uc *PaymentUseCase) GetPayables(ctx context.Context, companyID uuid.UUID) ([]domain.PayableBalance, error) {
	return uc.paymentRepo.GetPayables(ctx, companyID)
}
