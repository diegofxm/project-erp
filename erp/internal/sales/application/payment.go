package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type PaymentUseCase struct {
	paymentRepo domain.PaymentRepository
	saleRepo    domain.Repository
	bus         *events.Bus
}

func NewPaymentUseCase(paymentRepo domain.PaymentRepository, saleRepo domain.Repository, bus *events.Bus) *PaymentUseCase {
	return &PaymentUseCase{paymentRepo: paymentRepo, saleRepo: saleRepo, bus: bus}
}

type RecordPaymentRequest struct {
	SaleID        uuid.UUID `json:"sale_id"`
	PaymentDate   string    `json:"payment_date"` // YYYY-MM-DD
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Reference     string    `json:"reference"`
	Notes         string    `json:"notes"`
}

func (uc *PaymentUseCase) Record(ctx context.Context, companyID uuid.UUID, req RecordPaymentRequest) (*domain.SalePayment, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	// Verificar que la venta pertenece a la empresa y está confirmada
	sale, err := uc.saleRepo.GetByID(ctx, companyID, req.SaleID)
	if err != nil {
		return nil, err
	}
	if sale.Status != domain.StatusConfirmed {
		return nil, fmt.Errorf("solo se pueden registrar pagos sobre ventas confirmadas")
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

	p := domain.SalePayment{
		ID:            uuid.New(),
		CompanyID:     companyID,
		SaleID:        req.SaleID,
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

	uc.bus.Publish(domain.SalePaymentRecorded{
		PaymentID:     saved.ID,
		CompanyID:     companyID,
		SaleID:        saved.SaleID,
		Amount:        saved.Amount,
		PaymentMethod: saved.PaymentMethod,
		PaymentDate:   saved.PaymentDate,
	})

	return saved, nil
}

func (uc *PaymentUseCase) ListBySale(ctx context.Context, companyID, saleID uuid.UUID) ([]domain.SalePayment, error) {
	return uc.paymentRepo.ListBySale(ctx, companyID, saleID)
}

// GetReceivables devuelve las ventas confirmadas con saldo pendiente (cartera).
func (uc *PaymentUseCase) GetReceivables(ctx context.Context, companyID uuid.UUID) ([]domain.ReceivableBalance, error) {
	return uc.paymentRepo.GetReceivables(ctx, companyID)
}
